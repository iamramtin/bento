package confluent

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/linkedin/goavro/v2"

	"github.com/benthosdev/benthos/v4/internal/shutdown"
	"github.com/benthosdev/benthos/v4/public/service"
)

func schemaRegistryDecoderConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		// Stable(). TODO
		Categories("Parsing", "Integration").
		Summary("Automatically decodes and validates messages with schemas from a Confluent Schema Registry service.").
		Description(`
Decodes messages automatically from a schema stored within a [Confluent Schema Registry service](https://docs.confluent.io/platform/current/schema-registry/index.html) by extracting a schema ID from the message and obtaining the associated schema from the registry. If a message fails to match against the schema then it will remain unchanged and the error can be caught using error handling methods outlined [here](/docs/configuration/error_handling).

Currently only Avro schemas are supported.

### Avro JSON Format

This processor creates documents formatted as [Avro JSON](https://avro.apache.org/docs/current/specification/_print/#json-encoding) when decoding with Avro schemas. In this format the value of a union is encoded in JSON as follows:

- if its type is ` + "`null`, then it is encoded as a JSON `null`" + `;
- otherwise it is encoded as a JSON object with one name/value pair whose name is the type's name and whose value is the recursively encoded value. For Avro's named types (record, fixed or enum) the user-specified name is used, for other types the type name is used.

For example, the union schema ` + "`[\"null\",\"string\",\"Foo\"]`, where `Foo`" + ` is a record name, would encode:

- ` + "`null` as `null`" + `;
- the string ` + "`\"a\"` as `{\"string\": \"a\"}`" + `; and
- a ` + "`Foo` instance as `{\"Foo\": {...}}`, where `{...}` indicates the JSON encoding of a `Foo`" + ` instance.

However, it is possible to instead create documents in [standard/raw JSON format](https://pkg.go.dev/github.com/linkedin/goavro/v2#NewCodecForStandardJSONFull) by setting the field ` + "[`avro_raw_json`](#avro_raw_json) to `true`" + `.`).
		Field(service.NewBoolField("avro_raw_json").
			Description("Whether Avro messages should be decoded into normal JSON (\"json that meets the expectations of regular internet json\") rather than [Avro JSON](https://avro.apache.org/docs/current/specification/_print/#json-encoding). If `true` the schema returned from the subject should be decoded as [standard json](https://pkg.go.dev/github.com/linkedin/goavro/v2#NewCodecForStandardJSONFull) instead of as [avro json](https://pkg.go.dev/github.com/linkedin/goavro/v2#NewCodec). There is a [comment in goavro](https://github.com/linkedin/goavro/blob/5ec5a5ee7ec82e16e6e2b438d610e1cab2588393/union.go#L224-L249), the [underlining library used for avro serialization](https://github.com/linkedin/goavro), that explains in more detail the difference between the standard json and avro json.").
			Advanced().Default(false)).
		Field(service.NewStringField("url").Description("The base URL of the schema registry service.")).
		Field(service.NewObjectField("basic_auth",
			service.NewBoolField("enabled").
				Description("Whether to use basic authentication in requests.").
				Default(false),
			service.NewStringField("username").
				Description("Username required to authenticate.").
				Default(""),
			service.NewStringField("password").
				Description("Password required to authenticate.").
				Default("")).
			Advanced().
			Description("Allows you to specify basic authentication."),
		).Description("Enable basic authentication").
		Field(service.NewTLSField("tls"))
}

func init() {
	err := service.RegisterProcessor(
		"schema_registry_decode", schemaRegistryDecoderConfig(),
		func(conf *service.ParsedConfig, mgr *service.Resources) (service.Processor, error) {
			return newSchemaRegistryDecoderFromConfig(conf, mgr.Logger())
		})

	if err != nil {
		panic(err)
	}
}

//------------------------------------------------------------------------------

type schemaRegistryDecoder struct {
	client      *http.Client
	avroRawJSON bool

	schemaRegistryBaseURL        *url.URL
	schemaRegistryBasicAuthToken string

	schemas    map[int]*cachedSchemaDecoder
	cacheMut   sync.RWMutex
	requestMut sync.Mutex
	shutSig    *shutdown.Signaller

	logger *service.Logger
}

func newSchemaRegistryDecoderFromConfig(conf *service.ParsedConfig, logger *service.Logger) (*schemaRegistryDecoder, error) {
	urlStr, err := conf.FieldString("url")
	if err != nil {
		return nil, err
	}
	basicAuthEnabledBool, err := conf.FieldBool("basic_auth", "enabled")
	if err != nil {
		return nil, err
	}
	usernameStr, err := conf.FieldString("basic_auth", "username")
	if err != nil {
		return nil, err
	}
	passwordStr, err := conf.FieldString("basic_auth", "password")
	if err != nil {
		return nil, err
	}
	tlsConf, err := conf.FieldTLS("tls")
	if err != nil {
		return nil, err
	}
	avroRawJSON, err := conf.FieldBool("avro_raw_json")
	if err != nil {
		return nil, err
	}
	return newSchemaRegistryDecoder(urlStr, basicAuthEnabledBool, usernameStr, passwordStr, tlsConf, avroRawJSON, logger)
}

func newSchemaRegistryDecoder(urlStr string, basicAuthEnabledBool bool, usernameStr, passwordStr string, tlsConf *tls.Config, avroRawJSON bool, logger *service.Logger) (*schemaRegistryDecoder, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	var token string
	if basicAuthEnabledBool {
		token = base64.StdEncoding.EncodeToString([]byte(usernameStr + ":" + passwordStr))
	}

	s := &schemaRegistryDecoder{
		avroRawJSON:                  avroRawJSON,
		schemaRegistryBaseURL:        u,
		schemaRegistryBasicAuthToken: token,
		schemas:                      map[int]*cachedSchemaDecoder{},
		shutSig:                      shutdown.NewSignaller(),
		logger:                       logger,
	}

	s.client = http.DefaultClient
	if tlsConf != nil {
		s.client = &http.Client{}
		if c, ok := http.DefaultTransport.(*http.Transport); ok {
			cloned := c.Clone()
			cloned.TLSClientConfig = tlsConf
			s.client.Transport = cloned
		} else {
			s.client.Transport = &http.Transport{
				TLSClientConfig: tlsConf,
			}
		}
	}

	go func() {
		for {
			select {
			case <-time.After(schemaCachePurgePeriod):
				s.clearExpired()
			case <-s.shutSig.CloseAtLeisureChan():
				return
			}
		}
	}()
	return s, nil
}

func (s *schemaRegistryDecoder) Process(ctx context.Context, msg *service.Message) (service.MessageBatch, error) {
	b, err := msg.AsBytes()
	if err != nil {
		return nil, errors.New("unable to reference message as bytes")
	}

	id, remaining, err := extractID(b)
	if err != nil {
		return nil, err
	}

	decoder, err := s.getDecoder(id)
	if err != nil {
		return nil, err
	}

	msg.SetBytes(remaining)
	if err := decoder(msg); err != nil {
		return nil, err
	}

	return service.MessageBatch{msg}, nil
}

func (s *schemaRegistryDecoder) Close(ctx context.Context) error {
	s.shutSig.CloseNow()
	s.cacheMut.Lock()
	defer s.cacheMut.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	for k := range s.schemas {
		delete(s.schemas, k)
	}
	return nil
}

//------------------------------------------------------------------------------

type schemaDecoder func(m *service.Message) error

type cachedSchemaDecoder struct {
	lastUsedUnixSeconds int64
	decoder             schemaDecoder
}

func extractID(b []byte) (id int, remaining []byte, err error) {
	if len(b) == 0 {
		err = errors.New("message is empty")
		return
	}
	if b[0] != 0 {
		err = fmt.Errorf("serialization format version number %v not supported", b[0])
		return
	}
	id = int(binary.BigEndian.Uint32(b[1:5]))
	remaining = b[5:]
	return
}

const (
	schemaStaleAfter       = time.Minute * 10
	schemaCachePurgePeriod = time.Minute
)

func (s *schemaRegistryDecoder) clearExpired() {
	// First pass in read only mode to gather candidates
	s.cacheMut.RLock()
	targetTime := time.Now().Add(-schemaStaleAfter).Unix()
	var targets []int
	for k, v := range s.schemas {
		if atomic.LoadInt64(&v.lastUsedUnixSeconds) < targetTime {
			targets = append(targets, k)
		}
	}
	s.cacheMut.RUnlock()

	// Second pass fully locks schemas and removes stale decoders
	if len(targets) > 0 {
		s.cacheMut.Lock()
		for _, k := range targets {
			if s.schemas[k].lastUsedUnixSeconds < targetTime {
				delete(s.schemas, k)
			}
		}
		s.cacheMut.Unlock()
	}
}

func (s *schemaRegistryDecoder) getDecoder(id int) (schemaDecoder, error) {
	s.cacheMut.RLock()
	c, ok := s.schemas[id]
	s.cacheMut.RUnlock()
	if ok {
		atomic.StoreInt64(&c.lastUsedUnixSeconds, time.Now().Unix())
		return c.decoder, nil
	}

	s.requestMut.Lock()
	defer s.requestMut.Unlock()

	// We might've been beaten to making the request, so check once more whilst
	// within the request lock.
	s.cacheMut.RLock()
	c, ok = s.schemas[id]
	s.cacheMut.RUnlock()
	if ok {
		atomic.StoreInt64(&c.lastUsedUnixSeconds, time.Now().Unix())
		return c.decoder, nil
	}

	ctx, done := context.WithTimeout(context.Background(), time.Second*5)
	defer done()

	reqURL := *s.schemaRegistryBaseURL
	reqURL.Path = path.Join(reqURL.Path, fmt.Sprintf("/schemas/ids/%v", id))

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.schemaregistry.v1+json")

	if s.schemaRegistryBasicAuthToken != "" {
		req.Header.Add("Authorization", "Basic "+s.schemaRegistryBasicAuthToken)
	}

	var resBytes []byte
	for i := 0; i < 3; i++ {
		var res *http.Response
		if res, err = s.client.Do(req); err != nil {
			s.logger.Errorf("request failed for schema '%v': %v", id, err)
			continue
		}

		if res.StatusCode == http.StatusNotFound {
			err = fmt.Errorf("schema '%v' not found by registry", id)
			s.logger.Errorf(err.Error())
			break
		}

		if res.StatusCode != http.StatusOK {
			err = fmt.Errorf("request failed for schema '%v'", id)
			s.logger.Errorf(err.Error())
			// TODO: Best attempt at parsing out the body
			continue
		}

		if res.Body == nil {
			s.logger.Errorf("request for schema '%v' returned an empty body", id)
			err = errors.New("schema request returned an empty body")
			continue
		}

		resBytes, err = io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			s.logger.Errorf("failed to read response for schema '%v': %v", id, err)
			continue
		}

		break
	}
	if err != nil {
		return nil, err
	}

	resPayload := struct {
		Schema string `json:"schema"`
	}{}
	if err = json.Unmarshal(resBytes, &resPayload); err != nil {
		s.logger.Errorf("failed to parse response for schema '%v': %v", id, err)
		return nil, err
	}

	var codec *goavro.Codec
	if s.avroRawJSON {
		if codec, err = goavro.NewCodecForStandardJSONFull(resPayload.Schema); err != nil {
			s.logger.Errorf("failed to parse response for schema subject '%v': %v", id, err)
			return nil, err
		}
	} else {
		if codec, err = goavro.NewCodec(resPayload.Schema); err != nil {
			s.logger.Errorf("failed to parse response for schema subject '%v': %v", id, err)
			return nil, err
		}
	}

	decoder := func(m *service.Message) error {
		b, err := m.AsBytes()
		if err != nil {
			return err
		}

		native, _, err := codec.NativeFromBinary(b)
		if err != nil {
			return err
		}

		jb, err := codec.TextualFromNative(nil, native)
		if err != nil {
			return err
		}
		m.SetBytes(jb)

		return nil
	}

	s.cacheMut.Lock()
	s.schemas[id] = &cachedSchemaDecoder{
		lastUsedUnixSeconds: time.Now().Unix(),
		decoder:             decoder,
	}
	s.cacheMut.Unlock()

	return decoder, nil
}
