Error Handling
==============

## Processor Errors

Sometimes things can go wrong. Benthos supports a range of
[processors][processors] such as `http` and `lambda` that have the potential to
fail if their retry attempts are exhausted. When this happens the data is not
dropped but instead continues through the pipeline mostly unchanged. The content
remains the same but a metadata flag is added to the message that can be
referred to later in the pipeline using the
[`processor_failed`][processor_failed] condition.

This behaviour allows you to define in your config whether you would like the
failed messages to be dropped, recovered with more processing, or routed to a
dead-letter queue, or any combination thereof.

### Abandon on Failure

It's possible to define a list of processors which should be skipped for
messages that failed a previous stage using the [`try`][try] processor:

``` yaml
  - type: try
    try:
    - type: foo
    - type: bar # Skipped if foo failed
    - type: baz # Skipped if foo or bar failed
```

### Recover Failed Messages

Failed messages can be fed into their own processor steps with a
[`catch`][catch] processor:

``` yaml
  - type: catch
    catch:
    - type: foo # Recover here
```

Once messages finish the catch block they will have their failure flags removed
and are treated like regular messages. If this behaviour is not desired then it
is possible to simulate a catch block with a [`conditional`][conditional]
processor placed within a [`for_each`][for_each] processor:

``` yaml
  - type: for_each
    for_each:
    - type: conditional
      conditional:
        condition:
          type: processor_failed
        processors:
        - type: foo # Recover here
```

### Attempt Until Success

It's possible to reattempt a processor for a particular message until it is
successful with a [`while`][while] processor:

``` yaml
  - type: for_each
    for_each:
    - type: while
      while:
        at_least_once: true
        max_loops: 0 # Set this greater than zero to cap the number of attempts
        condition:
          type: processor_failed
        processors:
        - type: foo # Attempt this processor until success
```

This loop will block the pipeline and prevent the blocking message from being
acknowledged. It is therefore usually a good idea in practice to build your
condition with an exit strategy after N failed attempts so that the pipeline can
unblock itself without intervention.

### Drop Failed Messages

In order to filter out any failed messages from your pipeline you can simply use
a [`filter_parts`][filter_parts] processor:

``` yaml
  - type: filter_parts
    filter_parts:
      type: processor_failed
```

This will remove any failed messages from a batch.

### Route to a Dead-Letter Queue

It is possible to send failed messages to different destinations using either a
[`group_by`][group_by] processor with a [`switch`][switch] output, or a
[`broker`][broker] output with [`filter_parts`][filter_parts] processors.

``` yaml
pipeline:
  processors:
  - type: group_by
    group_by:
    - condition:
        type: processor_failed
output:
  type: switch
  switch:
    outputs:
    - output:
        type: foo # Dead letter queue
      condition:
        type: processor_failed
    - output:
        type: bar # Everything else
```

Note that the [`group_by`][group_by] processor is only necessary when messages
are batched.

Alternatively, using a `broker` output looks like this:

``` yaml
output:
  type: broker
  broker:
    pattern: fan_out
    outputs:
    - type: foo # Dead letter queue
      processors:
      - type: filter_parts
        filter_parts:
          type: processor_failed
    - type: bar # Everything else
      processors:
      - type: filter_parts
        filter_parts:
          type: not
          not:
            type: processor_failed
```

[processors]: ./processors/README.md
[processor_failed]: ./conditions/README.md#processor_failed
[filter_parts]: ./processors/README.md#filter_parts
[while]: ./processors/README.md#while
[for_each]: ./processors/README.md#for_each
[conditional]: ./processors/README.md#conditional
[catch]: ./processors/README.md#catch
[try]: ./processors/README.md#try
[group_by]: ./processors/README.md#group_by
[switch]: ./outputs/README.md#switch
[broker]: ./outputs/README.md#broker
