package kafkamod

import (
	"github.com/IBM/sarama"
)

// Handler has a call back which will receive message and function to mark offset of that massage.
// Do not move the code in handle to a goroutine because that will mess up the offset marking.
// The `Handle` itself is called within a goroutine for each partition claim in parallel.
// Returning error from handle will cause consumer to exit.
type Handler interface {
	Handle(msg *sarama.ConsumerMessage, mark func(metadata string)) error
}

// HandleFn can be used to pass function as Handler.
type HandleFn func(msg *sarama.ConsumerMessage, mark func(metadata string)) error

// Handle calls HandleFn.
func (h HandleFn) Handle(msg *sarama.ConsumerMessage, mark func(metadata string)) error {
	return h(msg, mark)
}

// ConcurrentGroupConsumer is convenience wrapper to consume partitions in parallel.
type ConcurrentGroupConsumer struct {
	handler Handler
	NoOpHandler
}

// NewConcurrentGroupConsumer wraps handler so that it implements sarama.ConsumerGroupHandler.
func NewConcurrentGroupConsumer(h Handler) *ConcurrentGroupConsumer {
	return &ConcurrentGroupConsumer{
		handler: h,
	}
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *ConcurrentGroupConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		msg := msg
		mark := func(metadata string) { session.MarkMessage(msg, metadata) }
		if err := c.handler.Handle(msg, mark); err != nil {
			return err
		}
	}

	return nil
}

// NoOpHandler can be used as embedded helper to avoid implementing methods of sarama.ConsumerGroupHandler which are not needed.
type NoOpHandler struct{}

// Setup does nothing.
func (NoOpHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup does nothing.
func (NoOpHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim does nothing.
func (NoOpHandler) ConsumeClaim(sarama.ConsumerGroupSession, sarama.ConsumerGroupClaim) error {
	return nil
}
