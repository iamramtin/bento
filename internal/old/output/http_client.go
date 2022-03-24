package output

import (
	"github.com/benthosdev/benthos/v4/internal/batch/policy"
	"github.com/benthosdev/benthos/v4/internal/component/metrics"
	"github.com/benthosdev/benthos/v4/internal/component/output"
	"github.com/benthosdev/benthos/v4/internal/docs"
	ihttpdocs "github.com/benthosdev/benthos/v4/internal/http/docs"
	"github.com/benthosdev/benthos/v4/internal/interop"
	"github.com/benthosdev/benthos/v4/internal/log"
	"github.com/benthosdev/benthos/v4/internal/old/output/writer"
)

func init() {
	Constructors[TypeHTTPClient] = TypeSpec{
		constructor: fromSimpleConstructor(NewHTTPClient),
		Summary: `
Sends messages to an HTTP server.`,
		Description: `
When the number of retries expires the output will reject the message, the
behaviour after this will depend on the pipeline but usually this simply means
the send is attempted again until successful whilst applying back pressure.

The URL and header values of this type can be dynamically set using function
interpolations described [here](/docs/configuration/interpolation#bloblang-queries).

The body of the HTTP request is the raw contents of the message payload. If the
message has multiple parts (is a batch) the request will be sent according to
[RFC1341](https://www.w3.org/Protocols/rfc1341/7_2_Multipart.html). This
behaviour can be disabled by setting the field ` + "[`batch_as_multipart`](#batch_as_multipart) to `false`" + `.

### Propagating Responses

It's possible to propagate the response from each HTTP request back to the input
source by setting ` + "`propagate_response` to `true`" + `. Only inputs that
support [synchronous responses](/docs/guides/sync_responses) are able to make use of
these propagated responses.`,
		Async:   true,
		Batches: true,
		Config: ihttpdocs.ClientFieldSpec(true,
			docs.FieldBool("batch_as_multipart", "Send message batches as a single request using [RFC1341](https://www.w3.org/Protocols/rfc1341/7_2_Multipart.html). If disabled messages in batches will be sent as individual requests.").Advanced(),
			docs.FieldBool("propagate_response", "Whether responses from the server should be [propagated back](/docs/guides/sync_responses) to the input.").Advanced(),
			docs.FieldInt("max_in_flight", "The maximum number of messages to have in flight at a given time. Increase this to improve throughput."),
			policy.FieldSpec(),
			docs.FieldObject(
				"multipart", "EXPERIMENTAL: Create explicit multipart HTTP requests by specifying an array of parts to add to the request, each part specified consists of content headers and a data field that can be populated dynamically. If this field is populated it will override the default request creation behaviour.",
			).Array().Advanced().HasDefault([]interface{}{}).WithChildren(
				docs.FieldInterpolatedString("content_type", "The content type of the individual message part.", "application/bin").HasDefault(""),
				docs.FieldInterpolatedString("content_disposition", "The content disposition of the individual message part.", `form-data; name="bin"; filename='${! meta("AttachmentName") }`).HasDefault(""),
				docs.FieldInterpolatedString("body", "The body of the individual message part.", `${! json("data.part1") }`).HasDefault(""),
			).AtVersion("3.63.0"),
		),
		Categories: []string{
			"Network",
		},
	}
}

// NewHTTPClient creates a new HTTPClient output type.
func NewHTTPClient(conf Config, mgr interop.Manager, log log.Modular, stats metrics.Type) (output.Streamed, error) {
	h, err := writer.NewHTTPClient(conf.HTTPClient, mgr, log, stats)
	if err != nil {
		return nil, err
	}
	w, err := NewAsyncWriter(TypeHTTPClient, conf.HTTPClient.MaxInFlight, h, log, stats)
	if err != nil {
		return w, err
	}
	if !conf.HTTPClient.BatchAsMultipart {
		w = OnlySinglePayloads(w)
	}
	return NewBatcherFromConfig(conf.HTTPClient.Batching, w, mgr, log, stats)
}
