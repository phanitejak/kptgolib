# What it is?

This is package having sarama.ConsumerGroup mock.

It is faking kafka partitions consuming allowing unit testing.

# How to use it?

1. In production code define following instead of using sarama.NewConsumerGroup directly:
   
    ```go
    // NewConsumerGroup is abstracting the sarama interfaces in order to do better unit testing.
    type NewConsumerGroup func(addrs []string, groupID string, config *sarama.Config) (sarama.ConsumerGroup, error)
    ```

2. In your init-code use sarama.NewConsumerGroup function as implementation of that interface.

3. In unit tests do following to initialize the mock :

    ```go
    config := mocks.NewTestConfig()
    cg, err := cgmocks.NewConsumerGroup([]string{"localhost"}, "testGroupID", config)
    assert.NoError(t, err)
    fnc := func(addrs []string, groupID string, config *sarama.Config) (sarama.ConsumerGroup, error) {
        return cg, nil
    }
    // TODO pass fnc to your production code init instead of sarama.NewConsumerGroup
    ```

4. In unit test first define which topic-partition to fake subscription to and then pass the messages you want to be read from:

    ```go
    claim1 := mockCG.YeldNewClaim(conf.Topics[0], 1, 0)
    claim1.YeldMessage4("{\"status\":\"ongoing\",\"operationId\":\"1234567890\"}")
    ```
    
    After that your code will receive this message in ConsumerGroupClaim.Messages() channel.
