Documentation
=============

## Core Components

- [Inputs](./inputs/README.md)
- [Buffers](./buffers/README.md)
- [Pipeline](./pipeline.md)
- [Outputs](./outputs/README.md)

## Other Components

- [Processors](./processors/README.md)
- [Conditions](./conditions/README.md)
- [Caches](./caches/README.md)
- [Rate Limits](./rate_limits/README.md)
- [Metrics](./metrics/README.md)
- [Tracers](./tracers/README.md)

## Other Sections

- [Core Concepts](./concepts.md) describes various Benthos concepts such as:
  - Configuration
  - Mutating And Filtering Content
  - Content Based Multiplexing
  - Sharing Resources Across Processors
  - Maximising IO Throughput
  - Maximising CPU Utilisation
- [Message Batching](./batching.md) explains how multiple part messages and
  message batching works within Benthos.
- [Error Handling](./error_handling.md) explains how you can handle errors from
  processor steps in order to recover or reroute the data.
- [Workflows](./workflows.md) explains how Benthos can be configured to easily
  support complex processor flows using automatic DAG resolution.
- [Making Configuration Easier](./configuration.md) explains some of the tools
  provided by Benthos that help make writing configs easier.
- [Config Interpolation](./config_interpolation.md) explains how to incorporate
  environment variables and dynamic values into your config files.
