//go:build integration
// +build integration

package kafkamod_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg/kafka"
	"gopkg/logging/loggingtest"
	"gopkg/runner/modules/kafkamod"
	"gopkg/tracing"
)

func noopOpt(*kafkamod.Consumer) error { return nil }

func TestIntegrationConsumer(t *testing.T) {
	t.Run("BasicNoPreAlloc", func(t *testing.T) { testIntegrationConsumer(t, noopOpt) })
	t.Run("BasicWithPreAlloc", func(t *testing.T) { testIntegrationConsumer(t, kafkamod.PreClaimPartitions()) })
	t.Run("ExitOnErrorInHandlerNoPreAlloc", func(t *testing.T) { testIntegrationExitOnErrorInHandler(t, noopOpt) })
	t.Run("ExitOnErrorInHandlerWithPreAlloc", func(t *testing.T) { testIntegrationExitOnErrorInHandler(t, kafkamod.PreClaimPartitions()) })
	t.Run("StickyRebalancNoPreAlloc", func(t *testing.T) { testIntegrationStickyRebalance(t, noopOpt) })
	t.Run("StickyRebalancWithPreAlloc", func(t *testing.T) { testIntegrationStickyRebalance(t, kafkamod.PreClaimPartitions()) })
}

func testIntegrationConsumer(t *testing.T, opt kafkamod.ConsumerOpt) {
	conf := kafkamod.ConsumerConfig{
		Brokers: []string{"127.0.0.1:9092"},
		Topics:  []string{"kafkamod-test-topic"},
		Group:   "kafkamod-test-group",
	}

	saramaConf := sarama.NewConfig()
	saramaConf.Version = sarama.V1_0_0_0
	saramaConf.Consumer.Offsets.Initial = sarama.OffsetOldest

	msgs := make(chan *sarama.ConsumerMessage)
	h := func(msg *sarama.ConsumerMessage, mark func(metadata string)) error {
		msgs <- msg
		mark("")
		return nil
	}

	c := kafkamod.NewConsumer(
		kafkamod.WithConsumerSaramaConfig(saramaConf),
		kafkamod.WithConsumerConfig(conf),
		kafkamod.WithConsumerHandler(kafkamod.NewConcurrentGroupConsumer(kafkamod.HandleFn(h))),
		opt,
	)

	err := c.Init(tracing.NewLogger(loggingtest.NewTestLogger(t)))
	require.NoError(t, err, "init failed")

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := c.Run()
		assert.NoError(t, err, "run failed")
	}()

	msg := sarama.ProducerMessage{
		Topic: conf.Topics[0],
		Key:   sarama.ByteEncoder("key"),
		Value: sarama.ByteEncoder("value"),
	}

	SendMsg(t, msg)

	select {
	case got := <-msgs:
		assert.Equal(t, []byte(msg.Key.(sarama.ByteEncoder)), got.Key)
		assert.Equal(t, []byte(msg.Value.(sarama.ByteEncoder)), got.Value)
	case <-time.After(time.Second * 5):
		t.Error("did not receive message on time")
	}

	require.NoError(t, c.Close(), "close failed")
	<-done
}

func testIntegrationExitOnErrorInHandler(t *testing.T, opt kafkamod.ConsumerOpt) {
	setEnv(t, "kafkamod-test-topic", "kafkamod-test-group")
	h := func(msg *sarama.ConsumerMessage, mark func(metadata string)) error {
		mark("")
		return errors.New("error in handler")
	}

	c := kafkamod.NewConsumer(
		kafkamod.WithConsumerEnvConfig(),
		kafkamod.WithConsumerHandler(kafkamod.NewConcurrentGroupConsumer(kafkamod.HandleFn(h))),
		opt,
	)

	err := c.Init(tracing.NewLogger(loggingtest.NewTestLogger(t)))
	require.NoError(t, err, "init failed")

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := c.Run()
		assert.NoError(t, err, "run failed")
	}()

	SendMsg(t, sarama.ProducerMessage{Topic: "kafkamod-test-topic"})

	select {
	case <-done:
	case <-time.After(time.Second * 5):
		t.Error("did not receive message on time")
	}

	require.NoError(t, c.Close())
}

func testIntegrationStickyRebalance(t *testing.T, opt kafkamod.ConsumerOpt) {
	setEnv(t, "kafkamod-sticky-test-topic", "kafkamod-sticky-test-group")

	conf := sarama.NewConfig()
	conf.Version = sarama.V1_0_0_0
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky

	h1 := newTestHandler(t, "h1")
	c1 := kafkamod.NewConsumer(
		kafkamod.WithConsumerEnvConfig(),
		kafkamod.WithConsumerHandler(h1),
		kafkamod.WithConsumerSaramaConfig(conf),
		kafkamod.WithConsumerMetricsPrefix("a"),
		opt,
	)

	err := c1.Init(tracing.NewLogger(loggingtest.NewTestLogger(t)))
	require.NoError(t, err, "init failed")

	done1 := make(chan struct{})
	go func() {
		defer close(done1)
		err := c1.Run()
		assert.NoError(t, err, "run failed")
	}()

	h1.AssertSetupCalled()
	h1.AssertConsumeClaimCalled()
	t.Log("first consumer running")

	c2 := kafkamod.NewConsumer(
		kafkamod.WithConsumerEnvConfig(),
		kafkamod.WithConsumerHandler(kafkamod.NoOpHandler{}),
		kafkamod.WithConsumerSaramaConfig(conf),
		kafkamod.WithConsumerMetricsPrefix("b"),
		opt,
	)
	err = c2.Init(tracing.NewLogger(loggingtest.NewTestLogger(t)))
	require.NoError(t, err, "init failed")

	done2 := make(chan struct{})
	go func() {
		defer close(done2)
		err := c2.Run()
		assert.NoError(t, err, "run failed")
	}()

	h1.AssertCleanupCalled()
	h1.AssertSetupCalled()
	h1.AssertConsumeClaimCalled()

	require.NoError(t, c1.Close(), "close failed")
	<-done1
	require.NoError(t, c2.Close(), "close failed")
	<-done2

	h1.AssertCleanupCalled()
}

func TestIntegrationNilHandler(t *testing.T) {
	c := kafkamod.NewConsumer()
	err := c.Init(tracing.NewLogger(loggingtest.NewTestLogger(t)))
	require.Error(t, err)
}

func TestIntegrationNoEnvSet(t *testing.T) {
	require.NoError(t, os.Unsetenv("KAFKA_CONSUMER_GROUP"))
	c := kafkamod.NewConsumer(kafkamod.WithConsumerEnvConfig())
	err := c.Init(tracing.NewLogger(loggingtest.NewTestLogger(t)))
	require.Error(t, err)
}

func TestIntegrationNoBrokers(t *testing.T) {
	c := kafkamod.NewConsumer(kafkamod.WithConsumerConfig(kafkamod.ConsumerConfig{}))
	err := c.Init(tracing.NewLogger(loggingtest.NewTestLogger(t)))
	require.Error(t, err)
}

func SendMsg(t testing.TB, msg sarama.ProducerMessage) {
	prod, err := kafka.NewProducerFromEnvWithPrefix("kafka_mod_test")
	require.NoError(t, err)
	err = prod.SendMessages(kafka.ProducerMessage{Ctx: context.Background(), Msg: &msg})
	require.NoError(t, err)
	err = prod.Close()
	require.NoError(t, err)
}

func setEnv(t testing.TB, topic, group string) {
	require.NoError(t, os.Setenv("KAFKA_CONSUMER_TOPICS", topic))
	require.NoError(t, os.Setenv("KAFKA_CONSUMER_GROUP", group))
	t.Cleanup(func() { assert.NoError(t, os.Unsetenv("KAFKA_CONSUMER_TOPICS")) })
	t.Cleanup(func() { assert.NoError(t, os.Unsetenv("KAFKA_CONSUMER_GROUP")) })
}

type testHandler struct {
	t                  testing.TB
	name               string
	setupCalled        chan struct{}
	cleanupCalled      chan struct{}
	consumeClaimCalled chan struct{}
}

func newTestHandler(t testing.TB, name string) *testHandler {
	return &testHandler{
		t:                  t,
		name:               name,
		setupCalled:        make(chan struct{}, 10),
		cleanupCalled:      make(chan struct{}, 10),
		consumeClaimCalled: make(chan struct{}, 10),
	}
}

func (h *testHandler) AssertSetupCalled() {
	select {
	case <-h.setupCalled:
	case <-time.After(time.Second * 30):
		h.t.Errorf("%s.Setup() not called in time", h.name)
	}
}

func (h *testHandler) AssertCleanupCalled() {
	select {
	case <-h.cleanupCalled:
	case <-time.After(time.Second * 30):
		h.t.Errorf("%s.Cleanup() not called in time", h.name)
	}
}

func (h *testHandler) AssertConsumeClaimCalled() {
	select {
	case <-h.consumeClaimCalled:
	case <-time.After(time.Second * 30):
		h.t.Errorf("%s.ConsumeClaim() not called in time", h.name)
	}
}

func (h *testHandler) Setup(sarama.ConsumerGroupSession) error {
	h.t.Logf("%s.Setup() called", h.name)
	h.setupCalled <- struct{}{}
	return nil
}

func (h *testHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.t.Logf("%s.Cleanup() called", h.name)
	h.cleanupCalled <- struct{}{}
	return nil
}

func (h *testHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.t.Logf("%s.ConsumeClaim() called", h.name)
	h.consumeClaimCalled <- struct{}{}
	for msg := range claim.Messages() {
		session.MarkMessage(msg, "")
	}
	return nil
}
