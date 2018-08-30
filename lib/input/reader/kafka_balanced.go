// Copyright (c) 2018 Ashley Jeffs
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package reader

import (
	"crypto/tls"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Jeffail/benthos/lib/log"
	"github.com/Jeffail/benthos/lib/message"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/types"
	btls "github.com/Jeffail/benthos/lib/util/tls"
	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
)

//------------------------------------------------------------------------------

// KafkaBalancedConfig contains configuration for the KafkaBalanced input type.
type KafkaBalancedConfig struct {
	Addresses       []string    `json:"addresses" yaml:"addresses"`
	ClientID        string      `json:"client_id" yaml:"client_id"`
	ConsumerGroup   string      `json:"consumer_group" yaml:"consumer_group"`
	CommitPeriodMS  int         `json:"commit_period_ms" yaml:"commit_period_ms"`
	Topics          []string    `json:"topics" yaml:"topics"`
	StartFromOldest bool        `json:"start_from_oldest" yaml:"start_from_oldest"`
	TargetVersion   string      `json:"target_version" yaml:"target_version"`
	TLS             btls.Config `json:"tls" yaml:"tls"`
}

// NewKafkaBalancedConfig creates a new KafkaBalancedConfig with default values.
func NewKafkaBalancedConfig() KafkaBalancedConfig {
	return KafkaBalancedConfig{
		Addresses:       []string{"localhost:9092"},
		ClientID:        "benthos_kafka_input",
		ConsumerGroup:   "benthos_consumer_group",
		CommitPeriodMS:  1000,
		Topics:          []string{"benthos_stream"},
		StartFromOldest: true,
		TargetVersion:   sarama.V1_0_0_0.String(),
		TLS:             btls.NewConfig(),
	}
}

//------------------------------------------------------------------------------

// KafkaBalanced is an input type that reads from a Kafka cluster by balancing
// partitions across other consumers of the same consumer group.
type KafkaBalanced struct {
	consumer *cluster.Consumer
	version  sarama.KafkaVersion
	cMut     sync.Mutex

	tlsConf *tls.Config

	offsetLastCommitted time.Time
	offsets             map[string]map[int32]int64

	mRcvErr     metrics.StatCounter
	mRebalanced metrics.StatCounter

	addresses []string
	topics    []string
	conf      KafkaBalancedConfig
	stats     metrics.Type
	log       log.Modular
}

// NewKafkaBalanced creates a new KafkaBalanced input type.
func NewKafkaBalanced(
	conf KafkaBalancedConfig, log log.Modular, stats metrics.Type,
) (*KafkaBalanced, error) {
	k := KafkaBalanced{
		conf:        conf,
		stats:       stats,
		mRcvErr:     stats.GetCounter("input.kafka_balanced.recv.error"),
		mRebalanced: stats.GetCounter("input.kafka_balanced.rebalanced"),
		offsets:     map[string]map[int32]int64{},
		log:         log.NewModule(".input.kafka_balanced"),
	}
	if conf.TLS.Enabled {
		var err error
		if k.tlsConf, err = conf.TLS.Get(); err != nil {
			return nil, err
		}
	}
	for _, addr := range conf.Addresses {
		for _, splitAddr := range strings.Split(addr, ",") {
			if len(splitAddr) > 0 {
				k.addresses = append(k.addresses, splitAddr)
			}
		}
	}
	for _, t := range conf.Topics {
		for _, splitTopics := range strings.Split(t, ",") {
			if len(splitTopics) > 0 {
				k.topics = append(k.topics, splitTopics)
			}
		}
	}
	var err error
	if k.version, err = sarama.ParseKafkaVersion(conf.TargetVersion); err != nil {
		return nil, err
	}
	return &k, nil
}

//------------------------------------------------------------------------------

// closeClients closes the kafka clients, this interrupts Read().
func (k *KafkaBalanced) closeClients() {
	k.cMut.Lock()
	defer k.cMut.Unlock()
	if k.consumer != nil {
		k.consumer.CommitOffsets()
		k.consumer.Close()

		// Drain all channels
		for range k.consumer.Messages() {
		}
		for range k.consumer.Notifications() {
		}
		for range k.consumer.Errors() {
		}

		k.consumer = nil
	}
}

//------------------------------------------------------------------------------

// Connect establishes a KafkaBalanced connection.
func (k *KafkaBalanced) Connect() error {
	k.cMut.Lock()
	defer k.cMut.Unlock()

	if k.consumer != nil {
		return nil
	}

	config := cluster.NewConfig()
	config.ClientID = k.conf.ClientID
	config.Net.DialTimeout = time.Second
	config.Version = k.version
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true
	config.Net.TLS.Enable = k.conf.TLS.Enabled
	if k.conf.TLS.Enabled {
		config.Net.TLS.Config = k.tlsConf
	}

	if k.conf.StartFromOldest {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	var consumer *cluster.Consumer
	var err error

	if consumer, err = cluster.NewConsumer(
		k.addresses,
		k.conf.ConsumerGroup,
		k.topics,
		config,
	); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case err, open := <-consumer.Errors():
				if !open {
					return
				}
				if err != nil {
					k.log.Errorf("KafkaBalanced message recv error: %v\n", err)
					k.mRcvErr.Incr(1)
				}
			case _, open := <-consumer.Notifications():
				if !open {
					return
				}
				k.mRebalanced.Incr(1)
			}
		}
	}()

	k.consumer = consumer
	k.log.Infof("Receiving KafkaBalanced messages from addresses: %s\n", k.addresses)
	return nil
}

func (k *KafkaBalanced) setOffset(topic string, partition int32, offset int64) {
	var topicMap map[int32]int64
	var exists bool
	if topicMap, exists = k.offsets[topic]; !exists {
		topicMap = map[int32]int64{}
		k.offsets[topic] = topicMap
	}
	topicMap[partition] = offset
}

// Read attempts to read a message from a KafkaBalanced topic.
func (k *KafkaBalanced) Read() (types.Message, error) {
	var consumer *cluster.Consumer

	k.cMut.Lock()
	if k.consumer != nil {
		consumer = k.consumer
	}
	k.cMut.Unlock()

	if consumer == nil {
		return nil, types.ErrNotConnected
	}

	data, open := <-consumer.Messages()
	if !open {
		k.closeClients()
		return nil, types.ErrNotConnected
	}

	msg := message.New([][]byte{data.Value})

	meta := msg.Get(0).Metadata()
	meta.Set("kafka_key", string(data.Key))
	meta.Set("kafka_partition", strconv.Itoa(int(data.Partition)))
	meta.Set("kafka_topic", data.Topic)
	meta.Set("kafka_offset", strconv.Itoa(int(data.Offset)))
	meta.Set("kafka_timestamp_unix", strconv.FormatInt(data.Timestamp.Unix(), 10))
	for _, hdr := range data.Headers {
		meta.Set(string(hdr.Key), string(hdr.Value))
	}

	k.setOffset(data.Topic, data.Partition, data.Offset)
	return msg, nil
}

// Acknowledge instructs whether the current offset should be committed.
func (k *KafkaBalanced) Acknowledge(err error) error {
	if err == nil {
		k.cMut.Lock()
		if k.consumer != nil {
			for topic, v := range k.offsets {
				for part, offset := range v {
					k.consumer.MarkPartitionOffset(topic, part, offset, "")
				}
			}
		}
		k.cMut.Unlock()
	}

	if time.Since(k.offsetLastCommitted) <
		(time.Millisecond * time.Duration(k.conf.CommitPeriodMS)) {
		return nil
	}

	var commitErr error
	k.cMut.Lock()
	if k.consumer != nil {
		commitErr = k.consumer.CommitOffsets()
	} else {
		commitErr = types.ErrNotConnected
	}
	k.cMut.Unlock()

	if commitErr == nil {
		k.offsetLastCommitted = time.Now()
	}

	return commitErr
}

// CloseAsync shuts down the KafkaBalanced input and stops processing requests.
func (k *KafkaBalanced) CloseAsync() {
	go k.closeClients()
}

// WaitForClose blocks until the KafkaBalanced input has closed down.
func (k *KafkaBalanced) WaitForClose(timeout time.Duration) error {
	return nil
}

//------------------------------------------------------------------------------
