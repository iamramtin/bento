package confluent

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Jeffail/benthos/v3/internal/shutdown"
	"github.com/Jeffail/benthos/v3/public/x/service"
	"github.com/linkedin/goavro/v2"
)

func schemaRegistryDecoderConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		// Stable(). TODO
		Categories("Parsing", "Integration").
		Summary("Automatically decodes and validates messages with schemas from a Confluent Schema Registry service.").
		Description(`
Decodes messages automatically from a schema stored within a [Confluent Schema Registry service](https://docs.confluent.io/platform/current/schema-registry/index.html) by extracting a schema ID from the message and obtaining the associated schema from the registry. If a message fails to match against the schema then it will remain unchanged and the error can be caught using error handling methods outlined [here](/docs/configuration/error_handling).

Currently only Avro schemas are supported.`).
		Field(service.NewStringField("url").Description("The base URL of the schema registry service.")).
		Field(service.NewTLSField("tls"))
}

func init() {
	err := service.RegisterProcessor(
		"schema_registry_decode", schemaRegistryDecoderConfig(),
		func(conf *service.ParsedConfig, mgr *service.Resources) (service.Processor, error) {
			urlStr, err := conf.FieldString("url")
			if err != nil {
				return nil, err
			}
			tlsConf, err := conf.FieldTLS("tls")
			if err != nil {
				return nil, err
			}
			return newSchemaRegistryDecoder(urlStr, tlsConf, mgr.Logger())
		})

	if err != nil {
		panic(err)
	}
}

//------------------------------------------------------------------------------

type schemaRegistryDecoder struct {
	surl   string
	client *http.Client

	schemas    map[int]*cachedSchemaDecoder
	cacheMut   sync.RWMutex
	requestMut sync.Mutex
	shutSig    *shutdown.Signaller

	logger *service.Logger
}

func newSchemaRegistryDecoder(urlStr string, tlsConf *tls.Config, logger *service.Logger) (*schemaRegistryDecoder, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}
	u.Path = "/schemas/ids/"

	s := &schemaRegistryDecoder{
		surl:    u.String() + "%v",
		schemas: map[int]*cachedSchemaDecoder{},
		shutSig: shutdown.NewSignaller(),
		logger:  logger,
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

func (s *schemaRegistryDecoder) Process(ctx context.Context, msg *service.Message) ([]*service.Message, error) {
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

	newMsg := msg.Copy()
	newMsg.SetBytes(remaining)
	if err := decoder(newMsg); err != nil {
		return nil, err
	}

	return []*service.Message{newMsg}, nil
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
		err = fmt.Errorf("serialization format version number %v not supported", uint8(b[0]))
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

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(s.surl, id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.schemaregistry.v1+json")

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

		resBytes, err = ioutil.ReadAll(res.Body)
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
	if codec, err = goavro.NewCodec(resPayload.Schema); err != nil {
		s.logger.Errorf("failed to parse response for schema '%v': %v", id, err)
		return nil, err
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
		m.SetStructured(native)
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
