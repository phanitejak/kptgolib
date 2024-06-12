// Package cgmocks offers sarama ConsumerGroup mocks
//
// Unit testing goes like this:
//
// In production code define following instead of using sarama.NewConsumerGroup directly:
// 1.
// 		// NewConsumerGroup is abstracting the sarama interfaces in order to do better unit testing.
// 		type NewConsumerGroup func(addrs []string, groupID string, config *sarama.Config) (sarama.ConsumerGroup, error)
//
// 2.
//		in init-code use sarama.NewConsumerGroup as implementation of that interface.
//
//
// 3.
// In unit tests do following to initialize  :
// 		config := mocks.NewTestConfig()
// 		cg, err := cgmocks.NewConsumerGroup([]string{"localhost"}, "testGroupID", config)
// 		assert.NoError(t, err)
// 		fnc := func(addrs []string, groupID string, config *sarama.Config) (sarama.ConsumerGroup, error) {
// 			return cg, nil
// 		}
//      pass fnc to your production code init instead of sarama.NewConsumerGroup
//
// 4.
//	In unit test first define which topic-partition to fake subscription to and then pass the messages you want to be read from:
// 		claim1 := mockCG.YeldNewClaim(conf.Topics[0], 1, 0)
// 		claim1.YeldMessage4("{\"status\":\"ongoing\",\"operationId\":\"1234567890\"}")
package cgmocks

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"go.uber.org/atomic"
)

// ConsumerGroupSession mock.
type ConsumerGroupSession struct {
	Ctx             context.Context
	GenerationIDVal int32
	MemberIDVal     string
	TopicPartitions map[string][]int32
	MarkedMessages  map[string][]*sarama.ConsumerMessage
	Committed       bool
}

// Claims returning the preconfigured topis.
func (m *ConsumerGroupSession) Claims() map[string][]int32 {
	return m.TopicPartitions
}

// MemberID is returning preconfigured member ID.
func (m *ConsumerGroupSession) MemberID() string {
	return m.MemberIDVal
}

// GenerationID is returning preconfigured generation ID.
func (m *ConsumerGroupSession) GenerationID() int32 {
	return m.GenerationIDVal
}

// MarkOffset is doing nothing.
func (m *ConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
	// TODO
}

// Commit is marking committed to true.
func (m *ConsumerGroupSession) Commit() {
	m.Committed = true
}

// ResetOffset does nothing.
func (m *ConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
	// TODO
}

// MarkMessage adds msg to MarkedMessages array.
func (m *ConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.MarkedMessages[metadata] = append(m.MarkedMessages[metadata], msg)
}

// Context returns assigned context.
func (m *ConsumerGroupSession) Context() context.Context {
	return m.Ctx
}

// ConsumerGroupClaim mock.
type ConsumerGroupClaim struct {
	TopicVal         string
	PartitionVal     int32
	InitialOffsetVal int64
	HighWaterMarkVal int64
	MessagesVal      chan *sarama.ConsumerMessage
	Offset           *atomic.Int64
}

// Topic returns the topic assigned via YeldNewClaim.
func (m *ConsumerGroupClaim) Topic() string {
	return m.TopicVal
}

// Partition returns the partition assigned via YeldNewClaim.
func (m *ConsumerGroupClaim) Partition() int32 {
	return m.PartitionVal
}

// InitialOffset returns the offset assigned via YeldNewClaim.
func (m *ConsumerGroupClaim) InitialOffset() int64 {
	return m.InitialOffsetVal
}

// HighWaterMarkOffset returns the mark assigned via YeldNewClaim.
func (m *ConsumerGroupClaim) HighWaterMarkOffset() int64 {
	return m.HighWaterMarkVal
}

// Messages returns the messagesd assigned via YeldMessage.
func (m *ConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return m.MessagesVal
}

// YeldMessage is pushing given message to Messages() channel.
func (m *ConsumerGroupClaim) YeldMessage(msg *sarama.ConsumerMessage) {
	msg.Offset = m.Offset.Add(1)
	msg.Partition = m.Partition()
	msg.Topic = m.Topic()
	m.MessagesVal <- msg
}

// YeldMessage2 is pushing given message to Messages() channel.
func (m *ConsumerGroupClaim) YeldMessage2(key, value []byte, headers []*sarama.RecordHeader) {
	msg := &sarama.ConsumerMessage{
		Headers: headers,
		Key:     key,
		Value:   value,
	}
	m.YeldMessage(msg)
}

// YeldMessage3 is pushing given message to Messages() channel.
func (m *ConsumerGroupClaim) YeldMessage3(key, value []byte) {
	msg := &sarama.ConsumerMessage{
		Key:   key,
		Value: value,
	}
	m.YeldMessage(msg)
}

// YeldMessage4 is pushing given message to Messages() channel.
func (m *ConsumerGroupClaim) YeldMessage4(value string) {
	msg := &sarama.ConsumerMessage{
		Value: []byte(value),
	}
	m.YeldMessage(msg)
}

// ConsumerGroup mock.
type ConsumerGroup struct {
	Addrs   []string
	GroupID string
	// Config is used in Claim's channel size of Messages()
	Config *sarama.Config
	// Err is used in Errors()
	Err chan error
	// Claims are used in Consume()
	Claims []*ConsumerGroupClaim
	// Session is used in Consume()
	Session *ConsumerGroupSession
	// LeaveKafka is indicating that Consume() has to exit and handler's Cleanup() be called
	LeaveKafka chan struct{}
}

// Consume is running Setup->ConsumeClaim->Cleanup sequence to assigned handler.
func (m *ConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	m.Session.Ctx = ctx
	fmt.Printf("mock consumer setup ...\n")
	if err := handler.Setup(m.Session); err != nil {
		return err
	}
	for _, t := range topics {
		for _, c := range m.Claims {
			if c.TopicVal == t {
				fmt.Printf("mock consumer claim for topic:%s and partition:%d\n", t, c.Partition())
				if err := handler.ConsumeClaim(m.Session, c); err != nil {
					return err
				}
			}
		}
	}
	fmt.Printf("mock consumer waiting for context close or Leavekafka message ...\n")
	select {
	case <-m.LeaveKafka:
	case <-ctx.Done():
	}
	fmt.Printf("mock consumer cleanup ...\n")
	return handler.Cleanup(m.Session)
}

// Errors returns Err channel.
func (m *ConsumerGroup) Errors() <-chan error {
	return m.Err
}

// Close does nothing.
func (m *ConsumerGroup) Close() error {
	return nil
}

// YeldNewClaim is defining what claims will be Consume()-ed.
func (m *ConsumerGroup) YeldNewClaim(topic string, partition int32, initialOffset int64) *ConsumerGroupClaim {
	claim := &ConsumerGroupClaim{
		TopicVal:         topic,
		PartitionVal:     partition,
		InitialOffsetVal: initialOffset,
		HighWaterMarkVal: 0,
		MessagesVal:      make(chan *sarama.ConsumerMessage, m.Config.ChannelBufferSize),
		Offset:           atomic.NewInt64(initialOffset),
	}
	m.Claims = append(m.Claims, claim)
	arr, flg := m.Session.TopicPartitions[topic]
	if flg {
		m.Session.TopicPartitions[topic] = append(arr, partition)
	} else {
		m.Session.TopicPartitions[topic] = []int32{partition}
	}
	return claim
}

// NewConsumerGroup instantiates new ConsumerGroup mock object.
func NewConsumerGroup(addrs []string, groupID string, config *sarama.Config) (*ConsumerGroup, error) {
	c := &ConsumerGroup{
		Addrs:   addrs,
		GroupID: groupID,
		Config:  config,
		Err:     make(chan error, 10),
		Claims:  []*ConsumerGroupClaim{},
		Session: &ConsumerGroupSession{
			TopicPartitions: map[string][]int32{},
			MarkedMessages:  map[string][]*sarama.ConsumerMessage{},
		},
	}
	fmt.Printf("mock consumer group %s create ...\n", groupID)
	return c, nil
}
