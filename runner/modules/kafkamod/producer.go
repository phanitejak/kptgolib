package kafkamod

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/hashicorp/go-multierror"
	"github.com/kelseyhightower/envconfig"
	"github.com/phanitejak/gopkg/metrics"
	"github.com/phanitejak/gopkg/tracing"
)

// ProducerMessage adds context to sarama.ProducerMessage.
type ProducerMessage struct {
	Ctx context.Context
	Msg *sarama.ProducerMessage
}

// ProducerOpt is a functional option type for Producer.
type ProducerOpt func(p *Producer) error

// WithProducerConfig allows setting producer configuration.
func WithProducerConfig(conf ProducerConfig) ProducerOpt {
	return func(p *Producer) error {
		p.conf = conf
		return nil
	}
}

// WithProducerSaramaConfig allows setting custom sarama configuration.
func WithProducerSaramaConfig(conf *sarama.Config) ProducerOpt {
	return func(p *Producer) error {
		p.saramaConf = conf
		return nil
	}
}

// WithProducerEnvConfig reads ProducerConfig from environment variables.
func WithProducerEnvConfig() ProducerOpt {
	return func(p *Producer) error {
		conf := ProducerConfig{}
		if err := envconfig.Process("", &conf); err != nil {
			return err
		}
		p.conf = conf
		return nil
	}
}

// WithProducerMetricsPrefix reads ProducerConfig from environment variables.
func WithProducerMetricsPrefix(prefix string) ProducerOpt {
	return func(p *Producer) error {
		p.conf.MetricsPrefix = prefix
		return nil
	}
}

// ProducerConfig allows passing configurations for producer.
type ProducerConfig struct {
	Brokers       []string `envconfig:"KAFKA_BROKERS" required:"true"`
	MetricsPrefix string   `envconfig:"KAFKA_PRODUCER_METRICS_PREFIX" default:"default"`
}

// Producer is a wrapper to use sarama.SyncProducer as module and adds tracing and metrics.
type Producer struct {
	client     sarama.SyncProducer
	conf       ProducerConfig
	saramaConf *sarama.Config
	done       chan struct{}
	opts       []ProducerOpt
}

// NewProducer creates producer with given options.
func NewProducer(opts ...ProducerOpt) *Producer {
	return &Producer{
		opts: opts,
	}
}

// Init applies the options, registers metrics and creates new sarama.SyncProducer.
func (p *Producer) Init(*tracing.Logger) error {
	p.done = make(chan struct{})
	p.saramaConf = sarama.NewConfig()
	p.saramaConf.Version = sarama.V1_0_0_0

	for _, opt := range p.opts {
		if err := opt(p); err != nil {
			return fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// These are required to be true for SyncProducer.
	p.saramaConf.Producer.Return.Successes = true
	p.saramaConf.Producer.Return.Errors = true

	err := metrics.CrossRegisterKafkaProducerMetricsPrefix(p.saramaConf.MetricRegistry, p.conf.MetricsPrefix)
	if err != nil {
		return fmt.Errorf("failed to register producer metrics: %w", err)
	}

	p.client, err = sarama.NewSyncProducer(p.conf.Brokers, p.saramaConf)
	if err != nil {
		return fmt.Errorf("failed to create sync producer: %w", err)
	}

	return nil
}

// Run blocks until Close is called.
func (p *Producer) Run() error {
	<-p.done
	return nil
}

// Close will unregister metrics and close the SyncProducer.
func (p *Producer) Close() error {
	defer close(p.done)

	var result *multierror.Error
	if err := unregisterProducer(p.conf.MetricsPrefix); err != nil {
		result = multierror.Append(result, err)
	}
	if err := p.client.Close(); err != nil {
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

// SendMessage produces a given message, and returns only when it either has
// succeeded or failed to produce. It will return the partition and the offset
// of the produced message, or an error if the message failed to produce.
func (p *Producer) SendMessage(ctx context.Context, msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	return p.client.SendMessage(tracing.MessageWithContext(ctx, msg))
}

// SendMessages produces a given set of messages, and returns only when all
// messages in the set have either succeeded or failed. Note that messages
// can succeed and fail individually; if some succeed and some fail,
// SendMessages will return an error.
func (p *Producer) SendMessages(msgs ...ProducerMessage) error {
	messages := make([]*sarama.ProducerMessage, len(msgs))
	for i, msg := range msgs {
		messages[i] = tracing.MessageWithContext(msg.Ctx, msg.Msg)
	}
	return p.client.SendMessages(messages)
}

// unregisterProducer will convert possible panic from UnregisterKafkaProducerMetricsPrefix into error.
func unregisterProducer(prefix string) (err error) {
	defer func() {
		if rErr := recover(); rErr != nil {
			err = fmt.Errorf("%s", rErr)
		}
	}()
	metrics.UnregisterKafkaProducerMetricsPrefix(prefix)
	return
}
