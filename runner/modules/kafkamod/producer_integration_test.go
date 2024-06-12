//go:build integration
// +build integration

package kafkamod_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/phanitejak/kptgolib/logging/loggingtest"
	"github.com/phanitejak/kptgolib/runner/modules/kafkamod"
	"github.com/phanitejak/kptgolib/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const prodTestTopic = "producer-mod-test"

func TestIntegrationProducer_SendMessage(t *testing.T) {
	r := newReceiver(t)
	p := kafkamod.NewProducer(
		kafkamod.WithProducerConfig(kafkamod.ProducerConfig{}),
		kafkamod.WithProducerEnvConfig(),
		kafkamod.WithProducerMetricsPrefix("my-prefix"),
	)
	log := tracing.NewLogger(loggingtest.NewTestLogger(t))
	require.NoError(t, p.Init(log))
	t.Cleanup(func() { assert.NoError(t, p.Close()) })

	errCh := make(chan error)
	go func() { errCh <- p.Run() }()

	const expectedMsg = "hello"
	_, _, err := p.SendMessage(context.Background(), &sarama.ProducerMessage{
		Topic: prodTestTopic,
		Value: sarama.StringEncoder(expectedMsg),
	})

	require.NoError(t, err)

	select {
	case gotMsg := <-r:
		assert.Equal(t, expectedMsg, string(gotMsg.Value))
	case <-time.After(time.Second):
		t.Fatal("Didn't receive message in time")
	}
}

func TestIntegrationProducer_SendMessages(t *testing.T) {
	r := newReceiver(t)
	p := kafkamod.NewProducer(
		kafkamod.WithProducerConfig(kafkamod.ProducerConfig{}),
		kafkamod.WithProducerEnvConfig(),
		kafkamod.WithProducerMetricsPrefix("my-prefix"),
	)

	log := tracing.NewLogger(loggingtest.NewTestLogger(t))
	require.NoError(t, p.Init(log))

	errCh := make(chan error)
	go func() { errCh <- p.Run() }()
	t.Cleanup(func() { assert.NoError(t, p.Close()) })

	const expectedMsg = "hello"
	err := p.SendMessages(
		kafkamod.ProducerMessage{
			Ctx: context.Background(),
			Msg: &sarama.ProducerMessage{
				Topic: prodTestTopic,
				Value: sarama.StringEncoder(expectedMsg),
			},
		},
	)

	require.NoError(t, err)

	select {
	case gotMsg := <-r:
		assert.Equal(t, expectedMsg, string(gotMsg.Value))
	case <-time.After(time.Second):
		t.Fatal("Didn't receive message in time")
	}
}

func TestIntegrationProducer_FailInit(t *testing.T) {
	p := kafkamod.NewProducer(
		kafkamod.WithProducerConfig(kafkamod.ProducerConfig{}),
		kafkamod.WithProducerSaramaConfig(sarama.NewConfig()),
	)

	log := tracing.NewLogger(loggingtest.NewTestLogger(t))
	require.Error(t, p.Init(log))
}

func newReceiver(t *testing.T) <-chan *sarama.ConsumerMessage {
	receive := make(chan *sarama.ConsumerMessage)
	c := kafkamod.NewConsumer(
		kafkamod.WithConsumerConfig(kafkamod.ConsumerConfig{
			Brokers: []string{os.Getenv("KAFKA_BROKERS")},
			Topics:  []string{prodTestTopic},
			Group:   "producer-mod-test-group",
		}),
		kafkamod.WithConsumerHandler(kafkamod.NewConcurrentGroupConsumer(
			kafkamod.HandleFn(func(msg *sarama.ConsumerMessage, mark func(metadata string)) error {
				mark("")
				receive <- msg
				return nil
			}),
		)),
	)

	log := tracing.NewLogger(loggingtest.NewTestLogger(t))

	require.NoError(t, c.Init(log))
	go func() { assert.NoError(t, c.Run()) }()
	t.Cleanup(func() { assert.NoError(t, c.Close()) })

	return receive
}
