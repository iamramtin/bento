package processor

import (
	"github.com/benthosdev/benthos/v4/internal/message"
	"github.com/benthosdev/benthos/v4/internal/tracing"
)

// MarkErr marks a message part as having failed. This includes modifying
// metadata to contain this error as well as adding the error to a tracing span
// if the message has one.
func MarkErr(part *message.Part, span *tracing.Span, err error) {
	if err == nil {
		return
	}
	part.MetaSet(message.FailFlagKey, err.Error())
	if span == nil {
		span = tracing.GetSpan(part)
	}
	if span != nil {
		span.SetTag("error", "true")
		span.LogKV(
			"event", "error",
			"type", err.Error(),
		)
	}
}

// GetFail returns an error string for a message part if it has failed, or an
// empty string if not.
func GetFail(part *message.Part) string {
	return part.MetaGet(message.FailFlagKey)
}
