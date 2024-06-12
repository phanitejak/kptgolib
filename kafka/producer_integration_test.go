//go:build integration
// +build integration

package kafka_test

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/phanitejak/kptgolib/kafka"
	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationProducer(t *testing.T) {
	const (
		msgCount = 5
		broker   = "127.0.0.1:9092"
		topic    = "producer-test"
		msgValue = "some test data"
	)
	closer, err := tracing.InitGlobalTracer()
	require.NoError(t, err)
	defer closer.Close()

	prod, err := kafka.NewDefaultProducer([]string{broker})
	require.NoError(t, err)

	msgs := make([]kafka.ProducerMessage, msgCount)
	for i := 0; i < msgCount; i++ {
		span, ctx := tracing.StartSpan("producer-test")
		msgs[i] = kafka.ProducerMessage{
			Ctx: ctx,
			Msg: &sarama.ProducerMessage{Topic: topic, Value: sarama.StringEncoder(msgValue)},
		}
		span.Finish()
	}

	err = prod.SendMessages(msgs...)
	require.NoError(t, err)

	conf := kafka.ConsumerConf{Brokers: []string{broker}, Topics: []string{topic}, Group: "producer-test-group"}
	consumer, err := kafka.NewConcurrentPartitionConsumer(conf, tracing.NewLogger(logging.NewLogger()))
	require.NoError(t, err)

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

type chReader chan *sarama.ConsumerMessage

func (ch chReader) handler(msg *sarama.ConsumerMessage, mark func(string)) error {
	ch <- msg
	mark("")
	return nil
}
