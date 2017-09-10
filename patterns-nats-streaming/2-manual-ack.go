package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
)

func logCloser(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Printf("close error: %s", err)
	}
}

func main() {
	conn, err := stan.Connect(
		"test-cluster",
		"test-client",
		stan.NatsURL("nats://localhost:4222"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer logCloser(conn)

	wg := &sync.WaitGroup{}

	sub, err := conn.Subscribe("counter", func(msg *stan.Msg) {
		// Add jitter..
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

		// Simulate failed acks to show redelivery behavior. 80% of the time
		// the ack will "succeed"
		if rand.Float32() > 0.2 {
			if err := msg.Ack(); err != nil {
				log.Print("failed to ack")
			}

			fmt.Printf("seq = %d [redelivered = %v, acked = true]\n", msg.Sequence, msg.Redelivered)
			wg.Done()
		} else {
			fmt.Printf("seq = %d [redelivered = %v, acked = false]\n", msg.Sequence, msg.Redelivered)
		}
	}, stan.SetManualAckMode(), stan.AckWait(time.Second))
	if err != nil {
		log.Fatal(err)
	}
	defer logCloser(sub)

	// Publish up to 10.
	for i := 0; i < 10; i++ {
		wg.Add(1)

		err := conn.Publish("counter", nil)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Wait until all messages have been processed.
	wg.Wait()
}
