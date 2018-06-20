Conditions
==========

This document was generated with `benthos --list-conditions`

Within the list of Benthos [processors][0] you will find the [filter][1]
processor, which applies a condition to every message and only propagates them
if the condition passes. Conditions themselves can modify ('not') and combine
('and', 'or') other conditions, and can therefore be used to create complex
filters.

The format of a condition is similar to other Benthos types:

``` yaml
condition:
  type: content
  content:
    operator: equals
    part: 0
    arg: hello world
```

And using boolean condition types we can combine multiple conditions together:

``` yaml
condition:
  type: and
  and:
  - type: content
    content:
      operator: contains
      arg: hello world
  - type: or
    or:
    - type: content
      content:
        operator: contains
        arg: foo
    - type: not
      not:
        type: content
        content:
          operator: contains
          arg: bar
```

The above example could be summarised as 'content contains "hello world" and
also either contains "foo" or does _not_ contain "bar"'.

Conditions can be extremely useful for creating filters on an output. By using a
fan out output broker with 'filter' processors on the brokered outputs it is
possible to build
[curated data streams](../concepts.md#content-based-multiplexing) that filter on
the content of each message.

### Reusing Conditions

Sometimes large chunks of logic are reused across processors, or nested multiple
times as branches of a larger condition. It is possible to avoid writing
duplicate condition configs by using the [resource condition][2].

### Contents

1. [`and`](#and)
2. [`content`](#content)
3. [`count`](#count)
4. [`jmespath`](#jmespath)
5. [`not`](#not)
6. [`or`](#or)
7. [`resource`](#resource)
8. [`static`](#static)
9. [`xor`](#xor)

## `and`

``` yaml
type: and
and: []
```

And is a condition that returns the logical AND of its children conditions.

## `content`

``` yaml
type: content
content:
  arg: ""
  operator: equals_cs
  part: 0
```

Content is a condition that checks the content of a message part against a
logical operator and an argument.

Available logical operators are:

### `equals_cs`

Checks whether the part equals the argument (case sensitive.)

### `equals`

Checks whether the part equals the argument under unicode case-folding (case
insensitive.)

### `contains_cs`

Checks whether the part contains the argument (case sensitive.)

### `contains`

Checks whether the part contains the argument under unicode case-folding (case
insensitive.)

### `prefix_cs`

Checks whether the part begins with the argument (case sensitive.)

### `prefix`

Checks whether the part begins with the argument under unicode case-folding
(case insensitive.)

### `suffix_cs`

Checks whether the part ends with the argument (case sensitive.)

### `suffix`

Checks whether the part ends with the argument under unicode case-folding (case
insensitive.)

### `regexp_partial`

Checks whether any section of the message part matches a regular expression (RE2
syntax).

### `regexp_exact`

Checks whether the message part exactly matches a regular expression (RE2
syntax).

## `count`

``` yaml
type: count
count:
  arg: 100
```

Counts messages starting from one, returning true until the counter reaches its
target, at which point it will return false and reset the counter. This
condition is useful when paired with the `read_until` input, as it can
be used to cut the input stream off once a certain number of messages have been
read.

It is worth noting that each discrete count condition will have its own counter.
Parallel processors containing a count condition will therefore count
independently. It is, however, possible to share the counter across processor
pipelines by defining the count condition as a resource.

## `jmespath`

``` yaml
type: jmespath
jmespath:
  part: 0
  query: ""
```

Parses a message part as a JSON blob and attempts to apply a JMESPath expression
to it, expecting a boolean response. If the response is true the condition
passes, otherwise it does not. Please refer to the
[JMESPath website](http://jmespath.org/) for information and tutorials regarding
the syntax of expressions.

For example, with the following config:

``` yaml
jmespath:
  part: 0
  query: a == 'foo'
```

If the initial jmespaths of part 0 were:

``` json
{
	"a": "foo"
}
```

Then the condition would pass.

JMESPath is traditionally used for mutating JSON jmespath, in order to do this
please instead use the [`jmespath`](../processors/README.md#jmespath)
processor.

## `not`

``` yaml
type: not
not: {}
```

Not is a condition that returns the opposite (NOT) of its child condition. The
body of a not object is the child condition, i.e. in order to express 'part 0
NOT equal to "foo"' you could have the following YAML config:

``` yaml
type: not
not:
  type: content
  content:
    operator: equal
    part: 0
    arg: foo
```

Or, the same example as JSON:

``` json
{
	"type": "not",
	"not": {
		"type": "content",
		"content": {
			"operator": "equal",
			"part": 0,
			"arg": "foo"
		}
	}
}
```

## `or`

``` yaml
type: or
or: []
```

Or is a condition that returns the logical OR of its children conditions.

## `resource`

``` yaml
type: resource
resource: ""
```

Resource is a condition type that runs a condition resource by its name. This
condition allows you to run the same configured condition resource in multiple
processors, or as a branch of another condition.

For example, let's imagine we have two outputs, one of which only receives
messages that satisfy a condition and the other receives the logical NOT of that
same condition. In this example we can save ourselves the trouble of configuring
the same condition twice by referring to it as a resource, like this:

``` yaml
output:
  type: broker
  broker:
    pattern: fan_out
    outputs:
    - type: foo
      foo:
        processors:
        - type: condition
          condition:
            type: resource
            resource: foobar
    - type: bar
      bar:
        processors:
        - type: condition
          condition:
            type: not
            not:
              type: resource
              resource: foobar
resources:
  conditions:
    foobar:
      type: content
      content:
        operator: equals_cs
        part: 1
        arg: filter me please
```

It is also worth noting that when conditions are used as resources in this way
they will only be executed once per message, regardless of how many times they
are referenced (unless the content is modified). Therefore, resource conditions
can act as a runtime optimisation as well as a config optimisation.

## `static`

``` yaml
type: static
static: true
```

Static is a condition that always resolves to the same static boolean value.

## `xor`

``` yaml
type: xor
xor: []
```

Xor is a condition that returns the logical XOR of its children conditions,
meaning it only resolves to true if _exactly_ one of its children conditions
resolves to true.

[0]: ../processors/README.md
[1]: ../processors/README.md#filter
[2]: #resource
