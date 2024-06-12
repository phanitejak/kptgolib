// Package kafkamod provides kafka consumer functionalities that can be used as module for runner.
package kafkamod

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"github.com/hashicorp/go-multierror"
	"github.com/kelseyhightower/envconfig"
	"github.com/phanitejak/kptgolib/metrics"
	"github.com/phanitejak/kptgolib/tracing"
)

// ConsumerConfig for consumer.
type ConsumerConfig struct {
	Brokers            []string `envconfig:"KAFKA_BROKERS" required:"true"`
	Topics             []string `envconfig:"KAFKA_CONSUMER_TOPICS" required:"true"`
	MetricsPrefix      string   `envconfig:"KAFKA_CONSUMER_METRICS_PREFIX" default:""`
	Group              string   `envconfig:"KAFKA_CONSUMER_GROUP" required:"true"`
	PreClaimPartitions bool     `envconfig:"KAFKA_CONSUMER_GROUP_PRE_CLAIM_PARTITIONS" default:"false"`
}

// ConsumerOpt for Consumer.
type ConsumerOpt func(s *Consumer) error

// WithConsumerSaramaConfig allows setting custom sarama configuration.
func WithConsumerSaramaConfig(conf *sarama.Config) ConsumerOpt {
	return func(c *Consumer) error {
		c.saramaConf = conf
		return nil
	}
}

// WithConsumerEnvConfig reads Config from environment variables.
func WithConsumerEnvConfig() ConsumerOpt {
	return func(c *Consumer) error {
		conf := ConsumerConfig{}
		if err := envconfig.Process("", &conf); err != nil {
			return err
		}
		c.conf = conf
		return nil
	}
}

// WithConsumerConfig allows setting custom config for Consumer.
func WithConsumerConfig(conf ConsumerConfig) ConsumerOpt {
	return func(c *Consumer) error {
		c.conf = conf
		return nil
	}
}

// WithConsumerHandler is mandatory option for setting handler.
func WithConsumerHandler(h sarama.ConsumerGroupHandler) ConsumerOpt {
	return func(c *Consumer) error {
		c.handler.handler = h
		return nil
	}
}

// WithConsumerMetricsPrefix allows setting prefix for metrics registry.
func WithConsumerMetricsPrefix(prefix string) ConsumerOpt {
	return func(c *Consumer) error {
		c.conf.MetricsPrefix = prefix
		return nil
	}
}

// PreClaimPartitions will claim partition already when Init() is called but only starts forwarding messages once Run() is called.
func PreClaimPartitions() ConsumerOpt {
	return func(c *Consumer) error {
		c.conf.PreClaimPartitions = true
		return nil
	}
}

// Consumer provides kafka group consumer as runnable module.
type Consumer struct {
	opts       []ConsumerOpt
	conf       ConsumerConfig
	saramaConf *sarama.Config

	client  sarama.ConsumerGroup
	handler *handlerWrapper

	errCh       chan error
	runFinished chan struct{}
	runOnce     *sync.Once
	ctx         context.Context
	cancel      func()
}

// NewConsumer populates Consumer with given options.
func NewConsumer(opts ...ConsumerOpt) *Consumer {
	return &Consumer{
		opts: opts,
	}
}

// Init sets reasonable default configs and then applies all given options.
func (c *Consumer) Init(l *tracing.Logger) error {
	c.errCh = make(chan error, 1)
	c.runFinished = make(chan struct{})
	c.runOnce = &sync.Once{}
	c.saramaConf = sarama.NewConfig()
	c.saramaConf.Version = sarama.V1_0_0_0
	c.saramaConf.Consumer.Offsets.Initial = sarama.OffsetOldest

	c.handler = &handlerWrapper{
		log:     l,
		handler: nil, // Handler has to be set using WithHandler Opt.
	}

	for _, opt := range c.opts {
		if err := opt(c); err != nil {
			return fmt.Errorf("failed to apply option: %w", err)
		}
	}

	if c.handler.handler == nil {
		return fmt.Errorf("message handler was not set")
	}

	prefix := c.conf.Group
	if c.conf.MetricsPrefix != "" {
		prefix += "_" + c.conf.MetricsPrefix
	}

	err := metrics.CrossRegisterKafkaConsumerMetricsPrefix(c.saramaConf.MetricRegistry, prefix)
	if err != nil {
		return fmt.Errorf("failed to register metrics for kafka consumer: %w", err)
	}

	client, err := sarama.NewConsumerGroup(c.conf.Brokers, c.conf.Group, c.saramaConf)
	if err != nil {
		return fmt.Errorf("failed to create consumer group client: %w", err)
	}

	c.client = client
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.handler.cancel = c.cancel
	c.handler.ready = make(chan struct{})

	if c.conf.PreClaimPartitions {
		go c.consume()
	}

	return nil
}

// Run starts consuming messages and returns only if Close is called or consuming messages fails.
func (c *Consumer) Run() error {
	defer close(c.runFinished)

	// Signal handler that it can start forwarding messages
	close(c.handler.ready)
	go c.consume()
	return <-c.errCh
}

func (c *Consumer) consume() {
	c.runOnce.Do(func() {
		c.errCh <- func() error {
			for {
				err := c.client.Consume(c.ctx, c.conf.Topics, c.handler)
				if err != nil {
					return fmt.Errorf("consumer exited with error: %w", err)
				}

				if errors.Is(c.ctx.Err(), context.Canceled) {
					return nil
				}
			}
		}()
	})
}

// Close will make consumer exit gracefully.
func (c *Consumer) Close() error {
	c.cancel()
	<-c.runFinished // Wait for Run to exit.

	prefix := c.conf.Group
	if c.conf.MetricsPrefix != "" {
		prefix += "_" + c.conf.MetricsPrefix
	}

	var result *multierror.Error
	if err := unregisterConsumer(prefix); err != nil {
		result = multierror.Append(result, err)
	}
	if err := c.client.Close(); err != nil {
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

type handlerWrapper struct {
	handler sarama.ConsumerGroupHandler
	log     *tracing.Logger
	ready   chan struct{}
	cancel  func()
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (h *handlerWrapper) Setup(session sarama.ConsumerGroupSession) error {
	return h.handler.Setup(session)
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited.
func (h *handlerWrapper) Cleanup(session sarama.ConsumerGroupSession) error {
	return h.handler.Cleanup(session)
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (h *handlerWrapper) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	l := h.log.WithFields(map[string]interface{}{
		"topic":         claim.Topic(),
		"partition":     claim.Partition(),
		"initialoffset": claim.InitialOffset(),
	})
	l.Info("claim received")

	// Start forwarding messages once h.ready is closed.
	<-h.ready

	if err := h.handler.ConsumeClaim(session, claim); err != nil {
		h.cancel()
		l.With("HighWaterMarkOffset", claim.HighWaterMarkOffset()).Errorf("ConsumeClaim exiting with error: %s", err)
		return err
	}

	l.With("HighWaterMarkOffset", claim.HighWaterMarkOffset()).Info("ConsumeClaim exiting")
	return nil
}

// unregisterConsumer will convert possible panic from UnregisterKafkaConsumerMetricsPrefix into error.
func unregisterConsumer(prefix string) (err error) {
	defer func() {
		if rErr := recover(); rErr != nil {
			err = fmt.Errorf("%s", rErr)
		}
	}()
	metrics.UnregisterKafkaConsumerMetricsPrefix(prefix)
	return
}
