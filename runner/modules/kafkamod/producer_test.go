package kafkamod_test

import (
	"os"
	"testing"

	"github.com/IBM/sarama"
	"github.com/phanitejak/kptgolib/metrics"
	"github.com/phanitejak/kptgolib/runner/modules/kafkamod"
	"github.com/stretchr/testify/assert"
)

func TestWithProducerEnvConfig_Fail(t *testing.T) {
	brokers := os.Getenv("KAFKA_BROEKRS")
	assert.NoError(t, os.Unsetenv("KAFKA_BROEKRS"))
	t.Cleanup(func() { assert.NoError(t, os.Setenv("KAFKA_BROEKRS", brokers)) })

	err := kafkamod.NewProducer(kafkamod.WithProducerEnvConfig()).Init(nil)
	assert.Error(t, err)
}

func TestMetricsRegistrationFailure(t *testing.T) {
	const prefix = "prefix"

	conf := sarama.NewConfig()
	err := metrics.CrossRegisterKafkaProducerMetricsPrefix(conf.MetricRegistry, prefix)
	assert.NoError(t, err)
	t.Cleanup(func() { metrics.UnregisterKafkaProducerMetricsPrefix(prefix) })

	err = kafkamod.NewProducer(kafkamod.WithProducerMetricsPrefix(prefix)).Init(nil)
	assert.Error(t, err)
}

func TestClientCreationFailure(t *testing.T) {
	err := kafkamod.NewProducer(
		kafkamod.WithProducerConfig(
			kafkamod.ProducerConfig{
				Brokers: []string{"not a broker"},
			},
		),
	).Init(nil)
	assert.Error(t, err)
}
