package kafka

import (
	"os"

	"github.com/IBM/sarama"
	"github.com/kelseyhightower/envconfig"
	"github.com/phanitejak/kptgolib/metrics"
	"github.com/phanitejak/kptgolib/tracing"
)

// AsyncProducer is a wrapper on the sarama async_producer.
type AsyncProducer struct {
	asyncProducer sarama.AsyncProducer
	log           *tracing.Logger
	prefix        string
}

const defaultPrefix = "default"

// NewAsyncProducerFromEnv creates AsyncProducer using broker values from environment and default sarama and metric configurations.
func NewAsyncProducerFromEnv(logger *tracing.Logger) (*AsyncProducer, error) {
	return NewAsyncProducerFromEnvWithPrefix(logger, defaultPrefix)
}

// NewDefaultAsyncProducer creates AsyncProducer with default sarama and metric configurations.
func NewDefaultAsyncProducer(logger *tracing.Logger, brokers []string) (*AsyncProducer, error) {
	return NewDefaultAsyncProducerWithPrefix(logger, brokers, defaultPrefix)
}

// NewAsyncProducerFromConfig creates AsyncProducer using given sarama configuration
func NewAsyncProducerFromConfig(logger *tracing.Logger, brokers []string, config *sarama.Config) (*AsyncProducer, error) {
	return NewAsyncProducerFromConfigWithPrefix(logger, brokers, config, defaultPrefix)
}

// NewAsyncProducerFromEnvWithPrefix creates AsyncProducer using given metric prefix, broker from env. and default sarama configurations
func NewAsyncProducerFromEnvWithPrefix(logger *tracing.Logger, prefix string) (*AsyncProducer, error) {
	conf := ProducerConf{}
	if err := envconfig.Process("", &conf); err != nil {
		return nil, err
	}
	return NewDefaultAsyncProducerWithPrefix(logger, conf.Brokers, prefix)
}

// NewDefaultAsyncProducerWithPrefix creates AsyncProducer using given brokers and metrics prefix and default sarama configuration.
func NewDefaultAsyncProducerWithPrefix(logger *tracing.Logger, brokers []string, prefix string) (*AsyncProducer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_1_0_0
	config.ClientID, _ = os.Hostname()
	return NewAsyncProducerFromConfigWithPrefix(logger, brokers, config, prefix)
}

// NewAsyncProducerFromConfigWithPrefix creates new async producer with the given configuration and metrics prefix. The sarama configurations (like batching, flushing etc) should be tuned according to the service needs. The values for config.Producer.Return.Successes and config.Producer.Return.Errors will be set to true for reading from successes and errors channels.
func NewAsyncProducerFromConfigWithPrefix(logger *tracing.Logger, brokers []string, config *sarama.Config, prefix string) (*AsyncProducer, error) {
	err := metrics.CrossRegisterKafkaProducerMetricsPrefix(config.MetricRegistry, prefix)
	if err != nil {
		return nil, err
	}

	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	asynchProducer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}
	a := &AsyncProducer{
		asyncProducer: asynchProducer,
		log:           logger,
		prefix:        prefix,
	}
	go a.handleProducerResponse()
	return a, nil
}

func (a *AsyncProducer) handleProducerResponse() {
	successChan := a.asyncProducer.Successes()
	errorChan := a.asyncProducer.Errors()

	// Process successes and errors until closed
	for successChan != nil && errorChan != nil {
		select {
		case _, ok := <-a.asyncProducer.Successes():
			if !ok {
				successChan = nil
				continue
			}
		case produceErr, ok := <-a.asyncProducer.Errors():
			if !ok {
				errorChan = nil
				continue
			}
			if produceErr != nil {
				a.log.Errorf("error in sending message %v", produceErr.Err)
			}
		}
	}
	a.log.Info("Stopped producer response reader")
}

// SendMessages send the list of messages.
func (a *AsyncProducer) SendMessages(msgs ...ProducerMessage) {
	for _, msg := range msgs {
		a.asyncProducer.Input() <- tracing.MessageWithContext(msg.Ctx, msg.Msg)
	}
}

// Close - Closes the kafka Producer.
func (a *AsyncProducer) Close() error {
	metrics.UnregisterKafkaProducerMetricsPrefix(a.prefix)
	return a.asyncProducer.Close()
}
