package kafka

import (
	"context"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/kelseyhightower/envconfig"
	"github.com/phanitejak/kptgolib/metrics"
	"github.com/phanitejak/kptgolib/tracing"
)

// ConsumerConf partition consumer client configuration.
type ConsumerConf struct {
	Brokers []string `envconfig:"KAFKA_BROKERS" required:"true"`
	Topics  []string `envconfig:"KAFKA_CONSUMER_TOPICS" required:"true"`
	Group   string   `envconfig:"KAFKA_CONSUMER_GROUP" required:"true"`

	RetryEnabled        bool `envconfig:"KAFKA_CONSUMER_RETRY_ENABLED" default:"true"`
	RetryWaitTimeoutSec int  `envconfig:"KAFKA_CONSUMER_RETRY_WAIT_TIMEOUT" default:"1"`
}

// HandlerFunc kafka message handler function signature.
type HandlerFunc func(msg *sarama.ConsumerMessage, mark func(metadata string)) error

// ConsumerGroupHandler interface to handle consumer group lifecyle callbacks.
type ConsumerGroupHandler interface {
	Setup(sarama.ConsumerGroupSession) error
	Cleanup(sarama.ConsumerGroupSession) error
}

// ConcurrentPartitionConsumer represnet partition consumer client.
type ConcurrentPartitionConsumer struct {
	client            sarama.ConsumerGroup
	consumerGroup     string
	topics            []string
	messageHandler    HandlerFunc
	log               *tracing.Logger
	cancelContext     context.CancelFunc
	cancelWaitGroup   *sync.WaitGroup
	clientMutex       *sync.Mutex
	conf              ConsumerConf
	config            *sarama.Config
	groupHandlerMutex *sync.RWMutex
	groupHandler      ConsumerGroupHandler
	runSetupMutex     *sync.Mutex
}

// NewConcurrentPartitionConsumerFromEnv initilize the partition consumer client.
func NewConcurrentPartitionConsumerFromEnv(logger *tracing.Logger) (*ConcurrentPartitionConsumer, error) {
	conf := ConsumerConf{}
	if err := envconfig.Process("", &conf); err != nil {
		return nil, err
	}
	return NewConcurrentPartitionConsumer(conf, logger)
}

// NewConcurrentPartitionConsumerWithConfigFromEnv initilize the partition consumer client with given sarama config.
func NewConcurrentPartitionConsumerWithConfigFromEnv(config *sarama.Config, logger *tracing.Logger) (*ConcurrentPartitionConsumer, error) {
	conf := ConsumerConf{}
	if err := envconfig.Process("", &conf); err != nil {
		return nil, err
	}
	return NewConcurrentPartitionConsumerWithConfig(conf, config, logger)
}

// NewConcurrentPartitionConsumer initilize the partition consumer client.
func NewConcurrentPartitionConsumer(conf ConsumerConf, logger *tracing.Logger) (*ConcurrentPartitionConsumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V1_0_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	return NewConcurrentPartitionConsumerWithConfig(conf, config, logger)
}

// NewConcurrentPartitionConsumerWithConfig initilize the partition consumer client with given sarama config.
func NewConcurrentPartitionConsumerWithConfig(conf ConsumerConf, config *sarama.Config, logger *tracing.Logger) (*ConcurrentPartitionConsumer, error) {
	c := &ConcurrentPartitionConsumer{
		consumerGroup:     conf.Group,
		topics:            conf.Topics,
		log:               logger,
		cancelWaitGroup:   &sync.WaitGroup{},
		clientMutex:       &sync.Mutex{},
		groupHandlerMutex: &sync.RWMutex{},
		conf:              conf,
		config:            config,
		runSetupMutex:     &sync.Mutex{},
	}
	if err := c.initializeConsumerGroupClient(); err != nil {
		return nil, err
	}
	c.crossRegisterKafkaConsumerMetrics()
	return c, nil
}

// SetConsumerGroupHandler ...
func (c *ConcurrentPartitionConsumer) SetConsumerGroupHandler(groupHandler ConsumerGroupHandler) {
	c.groupHandlerMutex.Lock()
	defer c.groupHandlerMutex.Unlock()
	c.groupHandler = groupHandler
}

func (c *ConcurrentPartitionConsumer) crossRegisterKafkaConsumerMetrics() {
	err := metrics.CrossRegisterKafkaConsumerMetricsPrefix(c.config.MetricRegistry, c.consumerGroup)
	if err == nil {
		return
	}
	metrics.UnregisterKafkaConsumerMetricsPrefix(c.consumerGroup)
	err = metrics.CrossRegisterKafkaConsumerMetricsPrefix(c.config.MetricRegistry, c.consumerGroup)
	if err != nil {
		c.log.Infof("consumer group %s not registered with metrics", c.consumerGroup)
	}
}

func (c *ConcurrentPartitionConsumer) initializeConsumerGroupClient() error {
	c.clientMutex.Lock()
	defer c.clientMutex.Unlock()
	if c.client != nil {
		return nil
	}
	client, err := sarama.NewConsumerGroup(c.conf.Brokers, c.conf.Group, c.config)
	if err != nil {
		return err
	}

	c.client = client
	c.log.Debugf("consumer group client initialized for %s group", c.conf.Group)
	return nil
}

func (c *ConcurrentPartitionConsumer) closeConsumerGroupClient() {
	c.clientMutex.Lock()
	defer c.clientMutex.Unlock()
	if err := c.client.Close(); err != nil {
		c.log.Errorf("error closing consumer group client for %s group, %v", c.conf.Group, err)
	}
	c.log.Debugf("consumer group client closed for %s group", c.conf.Group)
	c.client = nil
}

// Run starts consumer group session and initialize the partition consumer cliams.
func (c *ConcurrentPartitionConsumer) Run(handler HandlerFunc) error {
	for {
		var err error
		if err = c.run(handler); err == nil {
			c.log.Info("exited consumer session")
			return nil
		}
		if !c.conf.RetryEnabled {
			return err
		}
		c.log.Infof("restarting consumer group client join %s group due to error: %v", c.conf.Group, err)
		time.Sleep(time.Duration(c.conf.RetryWaitTimeoutSec) * time.Second)
	}
}

func (c *ConcurrentPartitionConsumer) run(handler HandlerFunc) error {
	if err := c.initializeConsumerGroupClient(); err != nil {
		return err
	}

	c.cancelWaitGroup.Add(1)
	defer func() {
		c.closeConsumerGroupClient()
		c.cancelWaitGroup.Done()
	}()

	var ctx context.Context
	func() {
		c.runSetupMutex.Lock()
		defer c.runSetupMutex.Unlock()
		ctx, c.cancelContext = context.WithCancel(context.Background())
		c.messageHandler = handler
	}()

	for {
		if err := c.client.Consume(ctx, c.topics, c); err != nil {
			c.log.Errorf("error from consumer, %v", err)
			return err
		}

		// Check if the context was canceled.
		if ctx.Err() != nil {
			return nil
		}
	}
}

// Close Concurrent Partition Consumer.
func (c *ConcurrentPartitionConsumer) Close() {
	c.runSetupMutex.Lock()
	defer c.runSetupMutex.Unlock()

	c.cancelContext()
	c.cancelWaitGroup.Wait()
	metrics.UnregisterKafkaConsumerMetricsPrefix(c.consumerGroup)
	c.log.Debug("partition consumer closed for %s group", c.conf.Group)
}

// Setup Concurrent Partition Consumer initialization callback.
func (c *ConcurrentPartitionConsumer) Setup(session sarama.ConsumerGroupSession) error {
	c.log.Infof("setup consumer session, memberId: %s, generationId: %d, claims: %v", session.MemberID(), session.GenerationID(), session.Claims())
	c.groupHandlerMutex.RLock()
	defer c.groupHandlerMutex.RUnlock()
	if c.groupHandler != nil {
		_ = c.groupHandler.Setup(session)
	}
	return nil
}

// Cleanup Concurrent Partition Consumer cleanup callback.
func (c *ConcurrentPartitionConsumer) Cleanup(session sarama.ConsumerGroupSession) error {
	c.log.Infof("cleanup consumer session, memberId: %s, generationId: %d, claims: %v", session.MemberID(), session.GenerationID(), session.Claims())
	c.groupHandlerMutex.RLock()
	defer c.groupHandlerMutex.RUnlock()
	if c.groupHandler != nil {
		_ = c.groupHandler.Cleanup(session)
	}
	return nil
}

// ConsumeClaim Concurrent Partition Claim's message cosumer.
func (c *ConcurrentPartitionConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.log.Infof("consumer claim starting, topic: %s, partition: %d, initialOffset: %d", claim.Topic(), claim.Partition(), claim.InitialOffset())

	for msg := range claim.Messages() {
		msg := msg
		mark := func(metadata string) { session.MarkMessage(msg, metadata) }

		if err := c.messageHandler(msg, mark); err != nil {
			c.cancelContext()
			return err
		}
	}

	c.log.Infof("consumer claim exiting, topic: %s, partition: %d, initialOffset: %d", claim.Topic(), claim.Partition(), claim.InitialOffset())
	return nil
}
