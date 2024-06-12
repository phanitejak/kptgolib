package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/kelseyhightower/envconfig"
	"github.com/phanitejak/kptgolib/metrics"
	"github.com/phanitejak/kptgolib/tracing"
)

type ProducerConf struct {
	Brokers []string `envconfig:"KAFKA_BROKERS" required:"true" default:"kf-ckaf-kafka-headless.default.svc.cluster.local:9092"`
}

// Producer is a wrapper for sarama.SyncProducer which adds tracing and metrics automatically
// and has only single method for sending messages.
type Producer struct {
	client sarama.SyncProducer
	prefix string
}

type ProducerMessage struct {
	Ctx context.Context
	Msg *sarama.ProducerMessage
}

func NewDefaultProducer(brokers []string) (*Producer, error) {
	return NewDefaultProducerWithPrefix(brokers, "default")
}

// NewDefaultProducerWithPrefix creates new default kafka producer with given metrics prefix.
func NewDefaultProducerWithPrefix(brokers []string, prefix string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Version = sarama.V1_0_0_0

	return NewProducerFromConfigWithPrefix(brokers, config, prefix)
}

func NewProducerFromConfig(brokers []string, config *sarama.Config) (*Producer, error) {
	return NewProducerFromConfigWithPrefix(brokers, config, "default")
}

// NewProducerFromConfigWithPrefix creates new kafka producer with given metrics prefix.
func NewProducerFromConfigWithPrefix(brokers []string, config *sarama.Config, prefix string) (*Producer, error) {
	err := metrics.CrossRegisterKafkaProducerMetricsPrefix(config.MetricRegistry, prefix)
	if err != nil {
		return nil, err
	}

	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	p, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &Producer{client: p, prefix: prefix}, nil
}

func NewProducerFromEnv() (*Producer, error) {
	return NewProducerFromEnvWithPrefix("default")
}

// NewProducerFromEnvWithPrefix creates new kafka producer from env config with given metrics prefix.
func NewProducerFromEnvWithPrefix(prefix string) (*Producer, error) {
	conf := ProducerConf{}
	if err := envconfig.Process("", &conf); err != nil {
		return nil, err
	}
	return NewDefaultProducerWithPrefix(conf.Brokers, prefix)
}

func (p *Producer) SendMessages(msgs ...ProducerMessage) error {
	messages := make([]*sarama.ProducerMessage, len(msgs))
	for i, msg := range msgs {
		messages[i] = tracing.MessageWithContext(msg.Ctx, msg.Msg)
	}
	return p.client.SendMessages(messages)
}

func (p *Producer) Close() error {
	metrics.UnregisterKafkaProducerMetricsPrefix(p.prefix)
	return p.client.Close()
}
