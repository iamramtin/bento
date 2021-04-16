package output

import (
	"fmt"

	"github.com/Jeffail/benthos/v3/internal/component/output"
	"github.com/Jeffail/benthos/v3/internal/docs"
	"github.com/Jeffail/benthos/v3/internal/interop"
	"github.com/Jeffail/benthos/v3/lib/log"
	"github.com/Jeffail/benthos/v3/lib/metrics"
	"github.com/Jeffail/benthos/v3/lib/output/writer"
	"github.com/Jeffail/benthos/v3/lib/pipeline"
	"github.com/Jeffail/benthos/v3/lib/processor"
	"github.com/Jeffail/benthos/v3/lib/types"
	"github.com/Jeffail/benthos/v3/lib/util/config"
	"gopkg.in/yaml.v3"
)

//------------------------------------------------------------------------------

// Category describes the general category of an output.
type Category string

// Output categories
var (
	CategoryLocal    Category = "Local"
	CategoryAWS      Category = "AWS"
	CategoryGCP      Category = "GCP"
	CategoryAzure    Category = "Azure"
	CategoryServices Category = "Services"
	CategoryNetwork  Category = "Network"
	CategoryUtility  Category = "Utility"
)

// TypeSpec is a constructor and a usage description for each output type.
type TypeSpec struct {
	constructor ConstructorFunc

	// Async indicates whether this output benefits from sending multiple
	// messages asynchronously over the protocol.
	Async bool

	// Batches indicates whether this output benefits from batching of messages.
	Batches bool

	Status      docs.Status
	Summary     string
	Description string
	Categories  []Category
	Footnotes   string
	config      docs.FieldSpec
	FieldSpecs  docs.FieldSpecs
	Examples    []docs.AnnotatedExample
	Version     string
}

// AppendProcessorsFromConfig takes a variant arg of pipeline constructor
// functions and returns a new slice of them where the processors of the
// provided output configuration will also be initialized.
func AppendProcessorsFromConfig(
	conf Config,
	mgr types.Manager,
	log log.Modular,
	stats metrics.Type,
	pipelines ...types.PipelineConstructorFunc,
) []types.PipelineConstructorFunc {
	if len(conf.Processors) > 0 {
		pipelines = append(pipelines, []types.PipelineConstructorFunc{func(i *int) (types.Pipeline, error) {
			if i == nil {
				procs := 0
				i = &procs
			}
			processors := make([]types.Processor, len(conf.Processors))
			for j, procConf := range conf.Processors {
				pMgr, pLog, pMetrics := interop.LabelChild(fmt.Sprintf("processor.%v", *i), mgr, log, stats)
				var err error
				processors[j], err = processor.New(procConf, pMgr, pLog, pMetrics)
				if err != nil {
					return nil, fmt.Errorf("failed to create processor '%v': %v", procConf.Type, err)
				}
				*i++
			}
			return pipeline.NewProcessor(log, stats, processors...), nil
		}}...)
	}
	return pipelines
}

func fromSimpleConstructor(fn func(Config, types.Manager, log.Modular, metrics.Type) (Type, error)) ConstructorFunc {
	return func(
		conf Config,
		mgr types.Manager,
		log log.Modular,
		stats metrics.Type,
		pipelines ...types.PipelineConstructorFunc,
	) (Type, error) {
		output, err := fn(conf, mgr, log, stats)
		if err != nil {
			return nil, fmt.Errorf("failed to create output '%v': %w", conf.Type, err)
		}
		pipelines = AppendProcessorsFromConfig(conf, mgr, log, stats, pipelines...)
		return WrapWithPipelines(output, pipelines...)
	}
}

// ConstructorFunc is a func signature able to construct an output.
type ConstructorFunc func(Config, types.Manager, log.Modular, metrics.Type, ...types.PipelineConstructorFunc) (Type, error)

// WalkConstructors iterates each component constructor.
func WalkConstructors(fn func(ConstructorFunc, docs.ComponentSpec)) {
	for k, v := range Constructors {
		conf := v.config
		if len(v.FieldSpecs) > 0 {
			conf = docs.FieldComponent().WithChildren(v.FieldSpecs...)
		}
		spec := docs.ComponentSpec{
			Type:        docs.TypeOutput,
			Name:        k,
			Summary:     v.Summary,
			Description: v.Description,
			Footnotes:   v.Footnotes,
			Config:      conf,
			Examples:    v.Examples,
			Status:      v.Status,
			Version:     v.Version,
		}
		if len(v.Categories) > 0 {
			spec.Categories = make([]string, 0, len(v.Categories))
			for _, cat := range v.Categories {
				spec.Categories = append(spec.Categories, string(cat))
			}
		}
		spec.Description = output.Description(v.Async, v.Batches, spec.Description)
		fn(ConstructorFunc(v.constructor), spec)
	}
	for k, v := range pluginSpecs {
		spec := docs.ComponentSpec{
			Type:   docs.TypeOutput,
			Name:   k,
			Status: docs.StatusPlugin,
			Config: docs.FieldComponent().Unlinted(),
		}
		fn(ConstructorFunc(v.constructor), spec)
	}
}

// Constructors is a map of all output types with their specs.
var Constructors = map[string]TypeSpec{}

//------------------------------------------------------------------------------

// String constants representing each output type.
const (
	TypeAMQP               = "amqp"
	TypeAMQP09             = "amqp_0_9"
	TypeAMQP1              = "amqp_1"
	TypeAWSDynamoDB        = "aws_dynamodb"
	TypeAWSKinesis         = "aws_kinesis"
	TypeAWSKinesisFirehose = "aws_kinesis_firehose"
	TypeAWSS3              = "aws_s3"
	TypeAWSSNS             = "aws_sns"
	TypeAWSSQS             = "aws_sqs"
	TypeAzureBlobStorage   = "azure_blob_storage"
	TypeAzureQueueStorage  = "azure_queue_storage"
	TypeAzureTableStorage  = "azure_table_storage"
	TypeBlobStorage        = "blob_storage"
	TypeBroker             = "broker"
	TypeCache              = "cache"
	TypeCassandra          = "cassandra"
	TypeDrop               = "drop"
	TypeDropOn             = "drop_on"
	TypeDropOnError        = "drop_on_error"
	TypeDynamic            = "dynamic"
	TypeDynamoDB           = "dynamodb"
	TypeElasticsearch      = "elasticsearch"
	TypeFile               = "file"
	TypeFiles              = "files"
	TypeGCPCloudStorage    = "gcp_cloud_storage"
	TypeGCPPubSub          = "gcp_pubsub"
	TypeHDFS               = "hdfs"
	TypeHTTPClient         = "http_client"
	TypeHTTPServer         = "http_server"
	TypeInproc             = "inproc"
	TypeKafka              = "kafka"
	TypeKinesis            = "kinesis"
	TypeKinesisFirehose    = "kinesis_firehose"
	TypeMongoDB            = "mongodb"
	TypeMQTT               = "mqtt"
	TypeNanomsg            = "nanomsg"
	TypeNATS               = "nats"
	TypeNATSStream         = "nats_stream"
	TypeNSQ                = "nsq"
	TypePulsar             = "pulsar"
	TypeRedisHash          = "redis_hash"
	TypeRedisList          = "redis_list"
	TypeRedisPubSub        = "redis_pubsub"
	TypeRedisStreams       = "redis_streams"
	TypeReject             = "reject"
	TypeResource           = "resource"
	TypeRetry              = "retry"
	TypeS3                 = "s3"
	TypeSFTP               = "sftp"
	TypeSNS                = "sns"
	TypeSQL                = "sql"
	TypeSQS                = "sqs"
	TypeSTDOUT             = "stdout"
	TypeSubprocess         = "subprocess"
	TypeSwitch             = "switch"
	TypeSyncResponse       = "sync_response"
	TypeTableStorage       = "table_storage"
	TypeTCP                = "tcp"
	TypeTry                = "try"
	TypeUDP                = "udp"
	TypeSocket             = "socket"
	TypeWebsocket          = "websocket"
	TypeZMQ4               = "zmq4"
)

//------------------------------------------------------------------------------

// Config is the all encompassing configuration struct for all output types.
type Config struct {
	Label              string                         `json:"label" yaml:"label"`
	Type               string                         `json:"type" yaml:"type"`
	AMQP               writer.AMQPConfig              `json:"amqp" yaml:"amqp"`
	AMQP09             writer.AMQPConfig              `json:"amqp_0_9" yaml:"amqp_0_9"`
	AMQP1              writer.AMQP1Config             `json:"amqp_1" yaml:"amqp_1"`
	AWSDynamoDB        writer.DynamoDBConfig          `json:"aws_dynamodb" yaml:"aws_dynamodb"`
	AWSKinesis         writer.KinesisConfig           `json:"aws_kinesis" yaml:"aws_kinesis"`
	AWSKinesisFirehose writer.KinesisFirehoseConfig   `json:"aws_kinesis_firehose" yaml:"aws_kinesis_firehose"`
	AWSS3              writer.AmazonS3Config          `json:"aws_s3" yaml:"aws_s3"`
	AWSSNS             writer.SNSConfig               `json:"aws_sns" yaml:"aws_sns"`
	AWSSQS             writer.AmazonSQSConfig         `json:"aws_sqs" yaml:"aws_sqs"`
	AzureBlobStorage   writer.AzureBlobStorageConfig  `json:"azure_blob_storage" yaml:"azure_blob_storage"`
	AzureQueueStorage  writer.AzureQueueStorageConfig `json:"azure_queue_storage" yaml:"azure_queue_storage"`
	AzureTableStorage  writer.AzureTableStorageConfig `json:"azure_table_storage" yaml:"azure_table_storage"`
	BlobStorage        writer.AzureBlobStorageConfig  `json:"blob_storage" yaml:"blob_storage"`
	Broker             BrokerConfig                   `json:"broker" yaml:"broker"`
	Cache              writer.CacheConfig             `json:"cache" yaml:"cache"`
	Cassandra          CassandraConfig                `json:"cassandra" yaml:"cassandra"`
	Drop               writer.DropConfig              `json:"drop" yaml:"drop"`
	DropOn             DropOnConfig                   `json:"drop_on" yaml:"drop_on"`
	DropOnError        DropOnErrorConfig              `json:"drop_on_error" yaml:"drop_on_error"`
	Dynamic            DynamicConfig                  `json:"dynamic" yaml:"dynamic"`
	DynamoDB           writer.DynamoDBConfig          `json:"dynamodb" yaml:"dynamodb"`
	Elasticsearch      writer.ElasticsearchConfig     `json:"elasticsearch" yaml:"elasticsearch"`
	File               FileConfig                     `json:"file" yaml:"file"`
	Files              writer.FilesConfig             `json:"files" yaml:"files"`
	GCPCloudStorage    GCPCloudStorageConfig          `json:"gcp_cloud_storage" yaml:"gcp_cloud_storage"`
	GCPPubSub          writer.GCPPubSubConfig         `json:"gcp_pubsub" yaml:"gcp_pubsub"`
	HDFS               writer.HDFSConfig              `json:"hdfs" yaml:"hdfs"`
	HTTPClient         writer.HTTPClientConfig        `json:"http_client" yaml:"http_client"`
	HTTPServer         HTTPServerConfig               `json:"http_server" yaml:"http_server"`
	Inproc             InprocConfig                   `json:"inproc" yaml:"inproc"`
	Kafka              writer.KafkaConfig             `json:"kafka" yaml:"kafka"`
	Kinesis            writer.KinesisConfig           `json:"kinesis" yaml:"kinesis"`
	KinesisFirehose    writer.KinesisFirehoseConfig   `json:"kinesis_firehose" yaml:"kinesis_firehose"`
	MongoDB            MongoDBConfig                  `json:"mongodb" yaml:"mongodb"`
	MQTT               writer.MQTTConfig              `json:"mqtt" yaml:"mqtt"`
	Nanomsg            writer.NanomsgConfig           `json:"nanomsg" yaml:"nanomsg"`
	NATS               writer.NATSConfig              `json:"nats" yaml:"nats"`
	NATSStream         writer.NATSStreamConfig        `json:"nats_stream" yaml:"nats_stream"`
	NSQ                writer.NSQConfig               `json:"nsq" yaml:"nsq"`
	Plugin             interface{}                    `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Pulsar             PulsarConfig                   `json:"pulsar" yaml:"pulsar"`
	RedisHash          writer.RedisHashConfig         `json:"redis_hash" yaml:"redis_hash"`
	RedisList          writer.RedisListConfig         `json:"redis_list" yaml:"redis_list"`
	RedisPubSub        writer.RedisPubSubConfig       `json:"redis_pubsub" yaml:"redis_pubsub"`
	RedisStreams       writer.RedisStreamsConfig      `json:"redis_streams" yaml:"redis_streams"`
	Reject             RejectConfig                   `json:"reject" yaml:"reject"`
	Resource           string                         `json:"resource" yaml:"resource"`
	Retry              RetryConfig                    `json:"retry" yaml:"retry"`
	S3                 writer.AmazonS3Config          `json:"s3" yaml:"s3"`
	SFTP               SFTPConfig                     `json:"sftp" yaml:"sftp"`
	SNS                writer.SNSConfig               `json:"sns" yaml:"sns"`
	SQL                SQLConfig                      `json:"sql" yaml:"sql"`
	SQS                writer.AmazonSQSConfig         `json:"sqs" yaml:"sqs"`
	STDOUT             STDOUTConfig                   `json:"stdout" yaml:"stdout"`
	Subprocess         SubprocessConfig               `json:"subprocess" yaml:"subprocess"`
	Switch             SwitchConfig                   `json:"switch" yaml:"switch"`
	SyncResponse       struct{}                       `json:"sync_response" yaml:"sync_response"`
	TableStorage       writer.AzureTableStorageConfig `json:"table_storage" yaml:"table_storage"`
	TCP                writer.TCPConfig               `json:"tcp" yaml:"tcp"`
	Try                TryConfig                      `json:"try" yaml:"try"`
	UDP                writer.UDPConfig               `json:"udp" yaml:"udp"`
	Socket             writer.SocketConfig            `json:"socket" yaml:"socket"`
	Websocket          writer.WebsocketConfig         `json:"websocket" yaml:"websocket"`
	ZMQ4               *writer.ZMQ4Config             `json:"zmq4,omitempty" yaml:"zmq4,omitempty"`
	Processors         []processor.Config             `json:"processors" yaml:"processors"`
}

// NewConfig returns a configuration struct fully populated with default values.
func NewConfig() Config {
	return Config{
		Label:              "",
		Type:               "stdout",
		AMQP:               writer.NewAMQPConfig(),
		AMQP09:             writer.NewAMQPConfig(),
		AMQP1:              writer.NewAMQP1Config(),
		AWSDynamoDB:        writer.NewDynamoDBConfig(),
		AWSKinesis:         writer.NewKinesisConfig(),
		AWSKinesisFirehose: writer.NewKinesisFirehoseConfig(),
		AWSS3:              writer.NewAmazonS3Config(),
		AWSSNS:             writer.NewSNSConfig(),
		AWSSQS:             writer.NewAmazonSQSConfig(),
		AzureBlobStorage:   writer.NewAzureBlobStorageConfig(),
		AzureQueueStorage:  writer.NewAzureQueueStorageConfig(),
		AzureTableStorage:  writer.NewAzureTableStorageConfig(),
		BlobStorage:        writer.NewAzureBlobStorageConfig(),
		Broker:             NewBrokerConfig(),
		Cache:              writer.NewCacheConfig(),
		Cassandra:          NewCassandraConfig(),
		Drop:               writer.NewDropConfig(),
		DropOn:             NewDropOnConfig(),
		DropOnError:        NewDropOnErrorConfig(),
		Dynamic:            NewDynamicConfig(),
		DynamoDB:           writer.NewDynamoDBConfig(),
		Elasticsearch:      writer.NewElasticsearchConfig(),
		File:               NewFileConfig(),
		Files:              writer.NewFilesConfig(),
		GCPCloudStorage:    NewGCPCloudStorageConfig(),
		GCPPubSub:          writer.NewGCPPubSubConfig(),
		HDFS:               writer.NewHDFSConfig(),
		HTTPClient:         writer.NewHTTPClientConfig(),
		HTTPServer:         NewHTTPServerConfig(),
		Inproc:             NewInprocConfig(),
		Kafka:              writer.NewKafkaConfig(),
		Kinesis:            writer.NewKinesisConfig(),
		KinesisFirehose:    writer.NewKinesisFirehoseConfig(),
		MQTT:               writer.NewMQTTConfig(),
		MongoDB:            NewMongoDBConfig(),
		Nanomsg:            writer.NewNanomsgConfig(),
		NATS:               writer.NewNATSConfig(),
		NATSStream:         writer.NewNATSStreamConfig(),
		NSQ:                writer.NewNSQConfig(),
		Plugin:             nil,
		Pulsar:             NewPulsarConfig(),
		RedisHash:          writer.NewRedisHashConfig(),
		RedisList:          writer.NewRedisListConfig(),
		RedisPubSub:        writer.NewRedisPubSubConfig(),
		RedisStreams:       writer.NewRedisStreamsConfig(),
		Reject:             NewRejectConfig(),
		Resource:           "",
		Retry:              NewRetryConfig(),
		S3:                 writer.NewAmazonS3Config(),
		SFTP:               NewSFTPConfig(),
		SNS:                writer.NewSNSConfig(),
		SQL:                NewSQLConfig(),
		SQS:                writer.NewAmazonSQSConfig(),
		STDOUT:             NewSTDOUTConfig(),
		Subprocess:         NewSubprocessConfig(),
		Switch:             NewSwitchConfig(),
		SyncResponse:       struct{}{},
		TableStorage:       writer.NewAzureTableStorageConfig(),
		TCP:                writer.NewTCPConfig(),
		Try:                NewTryConfig(),
		UDP:                writer.NewUDPConfig(),
		Socket:             writer.NewSocketConfig(),
		Websocket:          writer.NewWebsocketConfig(),
		ZMQ4:               writer.NewZMQ4Config(),
		Processors:         []processor.Config{},
	}
}

//------------------------------------------------------------------------------

// SanitiseConfig returns a sanitised version of the Config, meaning sections
// that aren't relevant to behaviour are removed.
func SanitiseConfig(conf Config) (interface{}, error) {
	return conf.Sanitised(false)
}

// Sanitised returns a sanitised version of the config, meaning sections that
// aren't relevant to behaviour are removed. Also optionally removes deprecated
// fields.
func (conf Config) Sanitised(removeDeprecated bool) (interface{}, error) {
	outputMap, err := config.SanitizeComponent(conf)
	if err != nil {
		return nil, err
	}
	if spec, exists := pluginSpecs[conf.Type]; exists {
		if spec.confSanitiser != nil {
			outputMap["plugin"] = spec.confSanitiser(conf.Plugin)
		}
	}
	if err := docs.SanitiseComponentConfig(
		docs.TypeOutput,
		(map[string]interface{})(outputMap),
		docs.ShouldDropDeprecated(removeDeprecated),
	); err != nil {
		return nil, err
	}
	return outputMap, nil
}

//------------------------------------------------------------------------------

// UnmarshalYAML ensures that when parsing configs that are in a map or slice
// the default values are still applied.
func (conf *Config) UnmarshalYAML(value *yaml.Node) error {
	type confAlias Config
	aliased := confAlias(NewConfig())

	err := value.Decode(&aliased)
	if err != nil {
		return fmt.Errorf("line %v: %v", value.Line, err)
	}

	var spec docs.ComponentSpec
	if aliased.Type, spec, err = docs.GetInferenceCandidateFromNode(docs.TypeOutput, aliased.Type, value); err != nil {
		return fmt.Errorf("line %v: %w", value.Line, err)
	}

	if spec.Status == docs.StatusPlugin {
		if spec, exists := pluginSpecs[aliased.Type]; exists && spec.confConstructor != nil {
			confBytes, err := yaml.Marshal(aliased.Plugin)
			if err != nil {
				return fmt.Errorf("line %v: %v", value.Line, err)
			}

			conf := spec.confConstructor()
			if err = yaml.Unmarshal(confBytes, conf); err != nil {
				return fmt.Errorf("line %v: %v", value.Line, err)
			}
			aliased.Plugin = conf
		}
	} else {
		aliased.Plugin = nil
	}

	*conf = Config(aliased)
	return nil
}

//------------------------------------------------------------------------------

// New creates an output type based on an output configuration.
func New(
	conf Config,
	mgr types.Manager,
	log log.Modular,
	stats metrics.Type,
	pipelines ...types.PipelineConstructorFunc,
) (Type, error) {
	if mgrV2, ok := mgr.(interface {
		NewOutput(Config, ...types.PipelineConstructorFunc) (types.Output, error)
	}); ok {
		return mgrV2.NewOutput(conf, pipelines...)
	}
	if c, ok := Constructors[conf.Type]; ok {
		return c.constructor(conf, mgr, log, stats, pipelines...)
	}
	if c, ok := pluginSpecs[conf.Type]; ok {
		return c.constructor(conf, mgr, log, stats, pipelines...)
	}
	return nil, types.ErrInvalidOutputType
}

//------------------------------------------------------------------------------
