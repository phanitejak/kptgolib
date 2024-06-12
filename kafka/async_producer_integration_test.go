//go:build integration
// +build integration

package kafka_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/phanitejak/kptgolib/kafka"
	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	defaultBroker = os.Getenv("KAFKA_BROKERS")
	log           = tracing.NewLogger(logging.NewLogger())
)

func TestMain(m *testing.M) {
	closer, err := tracing.InitGlobalTracer()
	if err != nil {
		log.Fatalf("error trace init - %v", err)
	}

	exitCode := m.Run()
	_ = closer.Close()

	os.Exit(exitCode)
}

func TestIntegrationNonUniquePrefix(t *testing.T) {
	broker := "127.0.0.1:9092"
	prefix := "unique_check"

	config := sarama.NewConfig()
	prod, err := kafka.NewAsyncProducerFromConfigWithPrefix(log, []string{broker}, config, prefix)
	require.NoError(t, err)
	_, err = kafka.NewAsyncProducerFromConfigWithPrefix(log, []string{broker}, config, prefix)
	require.Error(t, err)
	_ = prod.Close()
}

func TestIntegrationUnreachableBroker(t *testing.T) {
	broker := "127.0.0.1:19039"

	log := tracing.NewLogger(logging.NewLogger())

	_, err := kafka.NewAsyncProducerFromConfigWithPrefix(log, []string{broker}, sarama.NewConfig(), "unreachable_broker")
	require.Error(t, err)
}

func TestIntegrationAsyncProducer(t *testing.T) {
	prod, err := kafka.NewAsyncProducerFromConfigWithPrefix(log, []string{defaultBroker}, sarama.NewConfig(), "async_producer_prefix")
	require.NoError(t, err)
	defer func() {
		_ = prod.Close()
	}()
	testAsyncProducerMessageSendReceive(t, prod)
}

func TestIntegrationNewAsyncProducerFromEnv(t *testing.T) {
	prod, err := kafka.NewAsyncProducerFromEnv(log)
	require.NoError(t, err)
	defer func() {
		_ = prod.Close()
	}()
	testAsyncProducerMessageSendReceive(t, prod)
}

func TestIntegrationNewDefaultAsyncProducer(t *testing.T) {
	prod, err := kafka.NewDefaultAsyncProducer(log, []string{defaultBroker})
	require.NoError(t, err)
	defer func() {
		_ = prod.Close()
	}()
	testAsyncProducerMessageSendReceive(t, prod)
}

func TestIntegrationNewAsyncProducerFromConfig(t *testing.T) {
	prod, err := kafka.NewAsyncProducerFromConfig(log, []string{defaultBroker}, sarama.NewConfig())
	require.NoError(t, err)
	defer func() {
		_ = prod.Close()
	}()
	testAsyncProducerMessageSendReceive(t, prod)
}

func TestIntegrationNewAsyncProducerFromEnvWithPrefix(t *testing.T) {
	prod, err := kafka.NewAsyncProducerFromEnvWithPrefix(log, "producer_from_env")
	require.NoError(t, err)
	defer func() {
		_ = prod.Close()
	}()
	testAsyncProducerMessageSendReceive(t, prod)
}

func TestIntegrationNewDefaultAsyncProducerWithPrefix(t *testing.T) {
	prod, err := kafka.NewDefaultAsyncProducerWithPrefix(log, []string{defaultBroker}, "producer_with_prefix")
	require.NoError(t, err)
	defer func() {
		_ = prod.Close()
	}()
	testAsyncProducerMessageSendReceive(t, prod)
}

func testAsyncProducerMessageSendReceive(t *testing.T, prod *kafka.AsyncProducer) {
	const (
		msgCount = 5
		topic    = "async-producer-test"
		msgValue = "message data"
	)

	msgs := make([]kafka.ProducerMessage, msgCount)
	for i := 0; i < msgCount; i++ {
		msgs[i] = kafka.ProducerMessage{
			Ctx: getTracingContext("async-producer-test"),
			Msg: &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder(msgValue)},
		}
	}

	prod.SendMessages(msgs...)

	conf := kafka.ConsumerConf{Brokers: []string{defaultBroker}, Topics: []string{topic}, Group: "async-producer-test-group"}
	consumer, err := kafka.NewConcurrentPartitionConsumer(conf, tracing.NewLogger(logging.NewLogger()))
	require.NoError(t, err)
	defer consumer.Close()

	c := 0
	msgCh := make(chReader)
	go func() {
		require.NoError(t, consumer.Run(msgCh.handler))
	}()

	for {
		select {
		case msg := <-msgCh:
			assert.Equal(t, msgValue, string(msg.Value), "unexpected value in message")
			assert.Equal(t, 2, len(msg.Headers), "message should have 1 header which contains tracing information")

			c++
			if c == msgCount {
				return
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("didn't receive messages in time, got: %d wanted: %d", len(msgs), msgCount)
		}
	}
}

func getTracingContext(opName string) context.Context {
	_, ctx := tracing.StartSpanFromContext(context.Background(), opName)
	return ctx
}
