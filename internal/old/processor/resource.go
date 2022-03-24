package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/benthosdev/benthos/v4/internal/component/metrics"
	"github.com/benthosdev/benthos/v4/internal/component/processor"
	"github.com/benthosdev/benthos/v4/internal/docs"
	"github.com/benthosdev/benthos/v4/internal/interop"
	"github.com/benthosdev/benthos/v4/internal/log"
	"github.com/benthosdev/benthos/v4/internal/message"
)

func init() {
	Constructors[TypeResource] = TypeSpec{
		constructor: NewResource,
		Categories: []string{
			"Utility",
		},
		Summary: `
Resource is a processor type that runs a processor resource identified by its label.`,
		Description: `
This processor allows you to reference the same configured processor resource in multiple places, and can also tidy up large nested configs. For example, the config:

` + "```yaml" + `
pipeline:
  processors:
    - bloblang: |
        root.message = this
        root.meta.link_count = this.links.length()
        root.user.age = this.user.age.number()
` + "```" + `

Is equivalent to:

` + "```yaml" + `
pipeline:
  processors:
    - resource: foo_proc

processor_resources:
  - label: foo_proc
    bloblang: |
      root.message = this
      root.meta.link_count = this.links.length()
      root.user.age = this.user.age.number()
` + "```" + `

You can find out more about resources [in this document.](/docs/configuration/resources)`,
		Config: docs.FieldString("", "").HasDefault(""),
	}
}

//------------------------------------------------------------------------------

// Resource is a processor that returns the result of a processor resource.
type Resource struct {
	mgr  interop.Manager
	name string
	log  log.Modular
}

// NewResource returns a resource processor.
func NewResource(
	conf Config, mgr interop.Manager, log log.Modular, stats metrics.Type,
) (processor.V1, error) {
	if !mgr.ProbeProcessor(conf.Resource) {
		return nil, fmt.Errorf("processor resource '%v' was not found", conf.Resource)
	}
	return &Resource{
		mgr:  mgr,
		name: conf.Resource,
		log:  log,
	}, nil
}

//------------------------------------------------------------------------------

// ProcessMessage applies the processor to a message, either creating >0
// resulting messages or a response to be sent back to the message source.
func (r *Resource) ProcessMessage(msg *message.Batch) (msgs []*message.Batch, res error) {
	if err := r.mgr.AccessProcessor(context.Background(), r.name, func(p processor.V1) {
		msgs, res = p.ProcessMessage(msg)
	}); err != nil {
		r.log.Errorf("Failed to obtain processor resource '%v': %v", r.name, err)
		return nil, err
	}
	return msgs, res
}

// CloseAsync shuts down the processor and stops processing requests.
func (r *Resource) CloseAsync() {
}

// WaitForClose blocks until the processor has closed down.
func (r *Resource) WaitForClose(timeout time.Duration) error {
	return nil
}
