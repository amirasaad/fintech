package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// RunSmokeTest produces and consumes messages on multiple topics
// to verify Kafka cluster functionality locally.
func RunSmokeTest() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	brokers := strings.TrimSpace(os.Getenv("BROKERS"))
	if brokers == "" {
		brokers = "localhost:9093,localhost:9092"
	}
	groupID := strings.TrimSpace(os.Getenv("GROUP_ID"))
	if groupID == "" {
		groupID = "fintech"
	}

	topics := []string{
		"fintech.events.test.event",
		"fintech.events.other.event",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create topics if they don't exist
	{
		dialer := &kafka.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(ctx, "tcp", strings.Split(brokers, ",")[0])
		if err != nil {
			logger.Error("dial failed", "error", err)
			return err
		}
		defer func() { _ = conn.Close() }()
		for _, t := range topics {
			err = conn.CreateTopics(kafka.TopicConfig{
				Topic:             t,
				NumPartitions:     1,
				ReplicationFactor: 1,
			})
			if err != nil && !strings.Contains(strings.ToLower(err.Error()), "already exists") {
				logger.Error("create topic failed", "topic", t, "error", err)
				return err
			}
			logger.Info("topic ready", "topic", t)
		}
	}

	// Produce messages
	w := &kafka.Writer{
		Addr:                   kafka.TCP(strings.Split(brokers, ",")...),
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		Balancer:               &kafka.Hash{},
	}
	defer func() { _ = w.Close() }()

	for i, t := range topics {
		err := w.WriteMessages(ctx, kafka.Message{
			Topic: t,
			Key:   []byte("key"),
			Value: []byte("message-" + time.Now().Format(time.RFC3339Nano)),
			Time:  time.Now(),
		})
		if err != nil {
			logger.Error("write failed", "topic", t, "error", err)
			return err
		}
		logger.Info("produced", "topic", t, "index", i)
	}

	// Consume messages
	for _, t := range topics {
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     strings.Split(brokers, ","),
			GroupID:     groupID,
			Topic:       t,
			StartOffset: kafka.FirstOffset,
			MinBytes:    1,
			MaxBytes:    10e6,
			MaxWait:     500 * time.Millisecond,
		})
		defer func(rd *kafka.Reader) { _ = rd.Close() }(r)

		readCtx, cancelRead := context.WithTimeout(ctx, 10*time.Second)
		defer cancelRead()

		msg, err := r.FetchMessage(readCtx)
		if err != nil {
			logger.Error("fetch failed", "topic", t, "error", err)
			return err
		}
		logger.Info("consumed", "topic", t, "value", string(msg.Value))
		_ = r.CommitMessages(ctx, msg)
	}

	logger.Info("kafka smoke test passed")
	return nil
}

// main runs the smoke test and exits non-zero on failure.
func main() {
	if err := RunSmokeTest(); err != nil {
		os.Exit(1)
	}
}
