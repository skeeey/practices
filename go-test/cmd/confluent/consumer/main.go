package main

import (
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const bootstrapServer = ""

func main() {

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        bootstrapServer,
		"group.id":                 "cluster1",
		"auto.offset.reset":        "earliest",
		"security.protocol":        "ssl",
		"ssl.ca.location":          "ca.crt",
		"ssl.certificate.location": "client.crt",
		"ssl.key.location":         "client.key",
	})

	if err != nil {
		panic(err)
	}

	c.SubscribeTopics([]string{"^sourceevents.*.cluster1", "^sourcebroadcast.*"}, nil)

	// A signal handler or similar could be used to set this to false to break the loop.
	run := true

	for run {
		msg, err := c.ReadMessage(time.Second)
		if err == nil {
			fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
		} else if !err.(kafka.Error).IsTimeout() {
			// The client will automatically try to recover from all errors.
			// Timeout is not considered an error because it is raised by
			// ReadMessage in absence of messages.
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
		}
	}

	c.Close()
}
