package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/Jeffail/benthos/v3/lib/log"
	"github.com/Jeffail/benthos/v3/lib/metrics"
	"github.com/Jeffail/benthos/v3/lib/output/writer"
	"github.com/go-redis/redis/v7"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ = registerIntegrationTest("redis", func(t *testing.T) {
	t.Parallel()

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	pool.MaxWait = time.Second * 30
	resource, err := pool.Run("redis", "latest", nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, pool.Purge(resource))
	})

	resource.Expire(900)
	require.NoError(t, pool.Retry(func() error {
		conf := writer.NewRedisStreamsConfig()
		conf.URL = fmt.Sprintf("tcp://localhost:%v", resource.GetPort("6379/tcp"))

		r, cErr := writer.NewRedisStreams(conf, log.Noop(), metrics.Noop())
		if cErr != nil {
			return cErr
		}
		cErr = r.Connect()

		r.CloseAsync()
		return cErr
	}))

	// STREAMS
	t.Run("streams", func(t *testing.T) {
		t.Parallel()
		template := `
output:
  redis_streams:
    url: tcp://localhost:$PORT
    stream: stream-$ID
    body_key: body
    max_length: 0
    max_in_flight: $MAX_IN_FLIGHT
    metadata:
      exclude_prefixes: [ $OUTPUT_META_EXCLUDE_PREFIX ]
    batching:
      count: $OUTPUT_BATCH_COUNT

input:
  redis_streams:
    url: tcp://localhost:$PORT
    body_key: body
    streams: [ stream-$ID ]
    limit: 10
    client_id: client-input-$ID
    consumer_group: group-$ID
`
		suite := integrationTests(
			integrationTestOpenClose(),
			integrationTestMetadata(),
			integrationTestMetadataFilter(),
			integrationTestSendBatch(10),
			integrationTestSendBatches(20, 100, 1),
			integrationTestStreamSequential(1000),
			integrationTestStreamParallel(1000),
			integrationTestStreamParallelLossy(1000),
			integrationTestStreamParallelLossyThroughReconnect(100),
			integrationTestSendBatchCount(10),
		)
		suite.Run(
			t, template,
			testOptSleepAfterInput(100*time.Millisecond),
			testOptSleepAfterOutput(100*time.Millisecond),
			testOptPort(resource.GetPort("6379/tcp")),
		)
		t.Run("with max in flight", func(t *testing.T) {
			t.Parallel()
			suite.Run(
				t, template,
				testOptSleepAfterInput(100*time.Millisecond),
				testOptSleepAfterOutput(100*time.Millisecond),
				testOptPort(resource.GetPort("6379/tcp")),
				testOptMaxInFlight(10),
			)
		})
	})

	t.Run("pubsub", func(t *testing.T) {
		t.Parallel()
		template := `
output:
  redis_pubsub:
    url: tcp://localhost:$PORT
    channel: channel-$ID
    max_in_flight: $MAX_IN_FLIGHT
    batching:
      count: $OUTPUT_BATCH_COUNT

input:
  redis_pubsub:
    url: tcp://localhost:$PORT
    channels: [ channel-$ID ]
`
		suite := integrationTests(
			integrationTestOpenClose(),
			integrationTestSendBatch(10),
			integrationTestSendBatches(20, 100, 1),
			integrationTestStreamSequential(100),
			integrationTestStreamParallel(100),
			integrationTestStreamParallelLossy(100),
			integrationTestSendBatchCount(10),
		)
		suite.Run(
			t, template,
			testOptSleepAfterInput(500*time.Millisecond),
			testOptSleepAfterOutput(500*time.Millisecond),
			testOptPort(resource.GetPort("6379/tcp")),
		)
		t.Run("with max in flight", func(t *testing.T) {
			t.Parallel()
			suite.Run(
				t, template,
				testOptSleepAfterInput(500*time.Millisecond),
				testOptSleepAfterOutput(500*time.Millisecond),
				testOptPort(resource.GetPort("6379/tcp")),
				testOptMaxInFlight(10),
			)
		})
	})

	t.Run("list", func(t *testing.T) {
		t.Parallel()
		template := `
output:
  redis_list:
    url: tcp://localhost:$PORT
    key: key-$ID
    max_in_flight: $MAX_IN_FLIGHT
    batching:
      count: $OUTPUT_BATCH_COUNT

input:
  redis_list:
    url: tcp://localhost:$PORT
    key: key-$ID
`
		suite := integrationTests(
			integrationTestOpenClose(),
			integrationTestSendBatch(10),
			integrationTestSendBatches(20, 100, 1),
			integrationTestStreamSequential(1000),
			integrationTestStreamParallel(1000),
			integrationTestStreamParallelLossy(1000),
			integrationTestSendBatchCount(10),
		)
		suite.Run(
			t, template,
			testOptSleepAfterInput(100*time.Millisecond),
			testOptSleepAfterOutput(100*time.Millisecond),
			testOptPort(resource.GetPort("6379/tcp")),
		)
		t.Run("with max in flight", func(t *testing.T) {
			t.Parallel()
			suite.Run(
				t, template,
				testOptSleepAfterInput(100*time.Millisecond),
				testOptSleepAfterOutput(100*time.Millisecond),
				testOptPort(resource.GetPort("6379/tcp")),
				testOptMaxInFlight(10),
			)
		})
	})

	// HASH
	t.Run("hash", func(t *testing.T) {
		t.Parallel()
		template := `
output:
  redis_hash:
    url: tcp://localhost:$PORT
    key: $ID-${! json("id") }
    fields:
      content: ${! content() }
`
		hashGetFn := func(env *testEnvironment, id string) (string, []string, error) {
			client := redis.NewClient(&redis.Options{
				Addr:    fmt.Sprintf("localhost:%v", resource.GetPort("6379/tcp")),
				Network: "tcp",
			})
			key := env.configVars.id + "-" + id
			res, err := client.HGet(key, "content").Result()
			if err != nil {
				return "", nil, err
			}
			return res, nil, nil
		}
		suite := integrationTests(
			integrationTestOutputOnlySendSequential(10, hashGetFn),
			integrationTestOutputOnlySendBatch(10, hashGetFn),
			integrationTestOutputOnlyOverride(hashGetFn),
		)
		suite.Run(
			t, template,
			testOptSleepAfterInput(100*time.Millisecond),
			testOptSleepAfterOutput(100*time.Millisecond),
			testOptPort(resource.GetPort("6379/tcp")),
		)
	})
})

var _ = registerIntegrationBench("redis", func(b *testing.B) {
	pool, err := dockertest.NewPool("")
	require.NoError(b, err)

	pool.MaxWait = time.Second * 30
	resource, err := pool.Run("redis", "latest", nil)
	require.NoError(b, err)
	b.Cleanup(func() {
		assert.NoError(b, pool.Purge(resource))
	})

	resource.Expire(900)
	require.NoError(b, pool.Retry(func() error {
		conf := writer.NewRedisStreamsConfig()
		conf.URL = fmt.Sprintf("tcp://localhost:%v", resource.GetPort("6379/tcp"))

		r, cErr := writer.NewRedisStreams(conf, log.Noop(), metrics.Noop())
		if cErr != nil {
			return cErr
		}
		cErr = r.Connect()

		r.CloseAsync()
		return cErr
	}))

	// STREAMS
	b.Run("streams", func(b *testing.B) {
		template := `
output:
  redis_streams:
    url: tcp://localhost:$PORT
    stream: stream-$ID
    body_key: body
    max_length: 0
    max_in_flight: $MAX_IN_FLIGHT
    metadata:
      exclude_prefixes: [ $OUTPUT_META_EXCLUDE_PREFIX ]

input:
  redis_streams:
    url: tcp://localhost:$PORT
    body_key: body
    streams: [ stream-$ID ]
    limit: 10
    client_id: client-input-$ID
    consumer_group: group-$ID
`
		suite := integrationBenchs(
			integrationBenchSend(20, 1),
			integrationBenchSend(10, 1),
			integrationBenchSend(1, 1),
			integrationBenchWrite(20),
			integrationBenchWrite(10),
			integrationBenchWrite(1),
		)
		suite.Run(
			b, template,
			testOptSleepAfterInput(100*time.Millisecond),
			testOptSleepAfterOutput(100*time.Millisecond),
			testOptPort(resource.GetPort("6379/tcp")),
		)
	})

	b.Run("pubsub", func(b *testing.B) {
		template := `
output:
  redis_pubsub:
    url: tcp://localhost:$PORT
    channel: channel-$ID
    max_in_flight: $MAX_IN_FLIGHT

input:
  redis_pubsub:
    url: tcp://localhost:$PORT
    channels: [ channel-$ID ]
`
		suite := integrationBenchs(
			integrationBenchSend(20, 1),
			integrationBenchSend(10, 1),
			integrationBenchSend(1, 1),
			integrationBenchWrite(20),
			integrationBenchWrite(10),
			integrationBenchWrite(1),
		)
		suite.Run(
			b, template,
			testOptSleepAfterInput(500*time.Millisecond),
			testOptSleepAfterOutput(500*time.Millisecond),
			testOptPort(resource.GetPort("6379/tcp")),
		)
	})

	b.Run("list", func(b *testing.B) {
		template := `
output:
  redis_list:
    url: tcp://localhost:$PORT
    key: key-$ID
    max_in_flight: $MAX_IN_FLIGHT

input:
  redis_list:
    url: tcp://localhost:$PORT
    key: key-$ID
`
		suite := integrationBenchs(
			integrationBenchSend(20, 1),
			integrationBenchSend(10, 1),
			integrationBenchSend(1, 1),
			integrationBenchWrite(20),
			integrationBenchWrite(10),
			integrationBenchWrite(1),
		)
		suite.Run(
			b, template,
			testOptSleepAfterInput(100*time.Millisecond),
			testOptSleepAfterOutput(100*time.Millisecond),
			testOptPort(resource.GetPort("6379/tcp")),
		)
	})
})
