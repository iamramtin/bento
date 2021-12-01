package nats

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"

	"github.com/Jeffail/benthos/v3/internal/impl/nats/auth"
	"github.com/Jeffail/benthos/v3/internal/shutdown"
	"github.com/Jeffail/benthos/v3/lib/input"
	"github.com/Jeffail/benthos/v3/public/service"
	"github.com/nats-io/nats.go"
)

func natsJetStreamInputConfig() *service.ConfigSpec {
	return service.NewConfigSpec().
		// Stable(). TODO
		Categories("Services").
		Version("3.46.0").
		Summary("Reads messages from NATS JetStream subjects.").
		Description(`
### Metadata

This input adds the following metadata fields to each message:

` + "```text" + `
- nats_subject
` + "```" + `

You can access these metadata fields using
[function interpolation](/docs/configuration/interpolation#metadata).

` + auth.Description()).
		Field(service.NewStringListField("urls").
			Description("A list of URLs to connect to. If an item of the list contains commas it will be expanded into multiple URLs.").
			Example([]string{"nats://127.0.0.1:4222"}).
			Example([]string{"nats://username:password@127.0.0.1:4222"})).
		Field(service.NewStringField("queue").
			Description("An optional queue group to consume as.").
			Optional()).
		Field(service.NewStringField("subject").
			Description("A subject to consume from. Supports wildcards for consuming multiple subjects.").
			Example("foo.bar.baz").Example("foo.*.baz").Example("foo.bar.*").Example("foo.>")).
		Field(service.NewStringField("durable").
			Description("Preserve the state of your consumer under a durable name.").
			Optional()).
		Field(service.NewStringAnnotatedEnumField("deliver", map[string]string{
			"all":  "Deliver all available messages.",
			"last": "Deliver starting with the last published messages.",
		}).
			Description("Determines which messages to deliver when consuming without a durable subscriber.").
			Default("all")).
		Field(service.NewIntField("max_ack_pending").
			Description("The maximum number of outstanding acks to be allowed before consuming is halted.").
			Advanced().
			Default(1024)).
		Field(service.NewTLSToggledField("tls")).
		Field(service.NewInternalField(auth.FieldSpec()))
}

func init() {
	err := service.RegisterInput(
		input.TypeNATSJetStream, natsJetStreamInputConfig(),
		func(conf *service.ParsedConfig, mgr *service.Resources) (service.Input, error) {
			return newJetStreamReaderFromConfig(conf, mgr.Logger())
		})

	if err != nil {
		panic(err)
	}
}

//------------------------------------------------------------------------------

type jetStreamReader struct {
	urls          string
	deliverOpt    nats.SubOpt
	subject       string
	queue         string
	durable       string
	maxAckPending int
	authConf      auth.Config
	tlsConf       *tls.Config

	log *service.Logger

	connMut  sync.Mutex
	natsConn *nats.Conn
	natsSub  *nats.Subscription

	shutSig *shutdown.Signaller
}

func newJetStreamReaderFromConfig(conf *service.ParsedConfig, log *service.Logger) (*jetStreamReader, error) {
	j := jetStreamReader{
		log:     log,
		shutSig: shutdown.NewSignaller(),
	}

	urlList, err := conf.FieldStringList("urls")
	if err != nil {
		return nil, err
	}
	j.urls = strings.Join(urlList, ",")

	deliver, err := conf.FieldString("deliver")
	if err != nil {
		return nil, err
	}
	switch deliver {
	case "all":
		j.deliverOpt = nats.DeliverAll()
	case "last":
		j.deliverOpt = nats.DeliverLast()
	default:
		return nil, fmt.Errorf("deliver option %v was not recognised", deliver)
	}

	if j.subject, err = conf.FieldString("subject"); err != nil {
		return nil, err
	}
	if conf.Contains("queue") {
		if j.queue, err = conf.FieldString("queue"); err != nil {
			return nil, err
		}
	}
	if conf.Contains("durable") {
		if j.durable, err = conf.FieldString("durable"); err != nil {
			return nil, err
		}
	}

	if j.maxAckPending, err = conf.FieldInt("max_ack_pending"); err != nil {
		return nil, err
	}

	tlsConf, tlsEnabled, err := conf.FieldTLSToggled("tls")
	if err != nil {
		return nil, err
	}
	if tlsEnabled {
		j.tlsConf = tlsConf
	}

	if j.authConf, err = AuthFromParsedConfig(conf.Namespace("auth")); err != nil {
		return nil, err
	}

	return &j, nil
}

//------------------------------------------------------------------------------

func (j *jetStreamReader) Connect(ctx context.Context) error {
	j.connMut.Lock()
	defer j.connMut.Unlock()

	if j.natsConn != nil {
		return nil
	}

	var natsConn *nats.Conn
	var natsSub *nats.Subscription
	var err error

	defer func() {
		if err != nil {
			if natsSub != nil {
				_ = natsSub.Drain()
			}
			if natsConn != nil {
				natsConn.Close()
			}
		}
	}()

	var opts []nats.Option
	if j.tlsConf != nil {
		opts = append(opts, nats.Secure(j.tlsConf))
	}
	opts = append(opts, auth.GetOptions(j.authConf)...)
	if natsConn, err = nats.Connect(j.urls, opts...); err != nil {
		return err
	}

	jCtx, err := natsConn.JetStream()
	if err != nil {
		return err
	}

	options := []nats.SubOpt{
		nats.ManualAck(),
	}
	if j.durable != "" {
		options = append(options, nats.Durable(j.durable))
	}
	options = append(options, j.deliverOpt)
	if j.maxAckPending != 0 {
		options = append(options, nats.MaxAckPending(j.maxAckPending))
	}

	if j.queue == "" {
		natsSub, err = jCtx.SubscribeSync(j.subject, options...)
	} else {
		natsSub, err = jCtx.QueueSubscribeSync(j.subject, j.queue, options...)
	}
	if err != nil {
		return err
	}

	j.log.Infof("Receiving NATS messages from JetStream subject: %v", j.subject)

	j.natsConn = natsConn
	j.natsSub = natsSub
	return nil
}

func (j *jetStreamReader) disconnect() {
	j.connMut.Lock()
	defer j.connMut.Unlock()

	if j.natsSub != nil {
		_ = j.natsSub.Drain()
		j.natsSub = nil
	}
	if j.natsConn != nil {
		j.natsConn.Close()
		j.natsConn = nil
	}
}

func (j *jetStreamReader) Read(ctx context.Context) (*service.Message, service.AckFunc, error) {
	j.connMut.Lock()
	natsSub := j.natsSub
	j.connMut.Unlock()
	if natsSub == nil {
		return nil, nil, service.ErrNotConnected
	}

	nmsg, err := natsSub.NextMsgWithContext(ctx)
	if err != nil {
		// TODO: Any errors need capturing here to signal a lost connection?
		return nil, nil, err
	}

	msg := service.NewMessage(nmsg.Data)
	msg.MetaSet("nats_subject", nmsg.Subject)

	return msg, func(ctx context.Context, res error) error {
		if res == nil {
			return nmsg.Ack()
		}
		return nmsg.Nak()
	}, nil
}

func (j *jetStreamReader) Close(ctx context.Context) error {
	go func() {
		j.disconnect()
		j.shutSig.ShutdownComplete()
	}()
	select {
	case <-j.shutSig.HasClosedChan():
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
