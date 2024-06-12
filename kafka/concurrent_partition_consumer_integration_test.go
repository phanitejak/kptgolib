//go:build integration
// +build integration

package kafka_test

import (
	"bytes"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/phanitejak/kptgolib/kafka"
	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMessage struct {
	GoroutineID uint64
	Message     []byte
}

func TestIntegrationConcurrentConsumption(t *testing.T) {
	const topic = "partition-concurrency-test"
	const consumerGroup = "concurrency-test"
	const kafkaBrokerAddr = "127.0.0.1:9092"
	const numberOfPartitions = 10
	const numberOfMessages = 25

	ensureNumberOfPartitions(t, kafkaBrokerAddr, topic, numberOfPartitions)

	msgChan := make(chan testMessage, 0)
	handlerFunc := func(msg *sarama.ConsumerMessage, mark func(string)) error {
		msgChan <- testMessage{GoroutineID: getGoroutineID(), Message: msg.Value}
		mark("")
		return nil
	}

	consumer, err := kafka.NewConcurrentPartitionConsumer(
		kafka.ConsumerConf{Brokers: []string{kafkaBrokerAddr}, Topics: []string{topic}, Group: consumerGroup},
		tracing.NewLogger(logging.NewLogger()))
	require.NoError(t, err)
	defer consumer.Close()

	go func() {
		require.NoError(t, consumer.Run(handlerFunc))
	}()

	// Produce numberOfMessages messages to the topics and distribute them to partitions in round-robin manner.
	// Expect that handlerFunc is called in as many goroutines as there are partitions.
	produceMessages(t, kafkaBrokerAddr, topic, numberOfMessages)
	receivedMessages, goroutineIDs := receiveMessages(msgChan, numberOfMessages)

	assert.Equal(t, numberOfMessages, len(receivedMessages), "Received %d messages instead of the expected %d", len(receivedMessages), numberOfMessages)
	assert.Equal(t, numberOfPartitions, len(goroutineIDs), "Number of utilized goroutines is %d instead of expected 10", len(goroutineIDs))
}

// This function is a hack. It extracts the goroutine id from a textual
// stack trace. The Go authors strongly advice against using goroutine IDs.
// Do not use this in production code.
func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func ensureNumberOfPartitions(t *testing.T, brokerAddr, topic string, numPartitions int32) {
	config := sarama.NewConfig()
	config.Version = sarama.V1_0_0_0

	broker := sarama.NewBroker(brokerAddr)
	broker.Open(config)
	defer broker.Close()

	// Try to create a topics with a specified number of partitions
	createTopic(t, broker, topic, numPartitions)

	// If the topics already exists, the above function call does not have effects.
	// Try to modify the number of partitions in the existing topics.
	createPartitions(t, broker, topic, numPartitions)
}

func createTopic(t *testing.T, broker *sarama.Broker, topic string, numPartitions int32) {
	_, err := broker.CreateTopics(&sarama.CreateTopicsRequest{
		TopicDetails: map[string]*sarama.TopicDetail{
			topic: {
				NumPartitions:     numPartitions,
				ReplicationFactor: 1,
			},
		},
		Timeout: 30 * time.Second,
	})
	require.NoError(t, err)
}

func createPartitions(t *testing.T, broker *sarama.Broker, topic string, numPartitions int32) {
	_, err := broker.CreatePartitions(&sarama.CreatePartitionsRequest{
		TopicPartitions: map[string]*sarama.TopicPartition{
			topic: {
				Count: numPartitions,
			},
		},
		Timeout: 30 * time.Second,
	})
	require.NoError(t, err)
}

func produceMessages(t *testing.T, brokerAddr, topic string, msgCount int) {
	config := sarama.NewConfig()
	config.Producer.Flush.MaxMessages = 1
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewRoundRobinPartitioner

	producer, err := sarama.NewSyncProducer([]string{brokerAddr}, config)
	require.NoError(t, err)
	defer producer.Close()

	msgs := make([]*sarama.ProducerMessage, 0, msgCount)
	for i := 0; i < msgCount; i++ {
		msgs = append(msgs, &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.StringEncoder(strconv.Itoa(i)),
		})
	}
	require.NoError(t, producer.SendMessages(msgs))
}

func receiveMessages(msgChan <-chan testMessage, msgCount int) (msgs []testMessage, goroutineIDs map[uint64]bool) {
	msgs = make([]testMessage, 0)
	goroutineIDs = make(map[uint64]bool)

	for {
		select {
		case msg := <-msgChan:
			msgs = append(msgs, msg)
			goroutineIDs[msg.GoroutineID] = true
			if len(msgs) >= msgCount {
				return
			}
		case <-time.After(10 * time.Second):
			return
		}
	}
	return
}
