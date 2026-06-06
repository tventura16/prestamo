// Package messaging envuelve segmentio/kafka-go con las piezas mínimas que
// el payment-service necesita: asegurar topics, publicar y consumir.
package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// EnsureTopics crea los topics si no existen (single-broker KRaft: 1 partición,
// RF 1). Es idempotente: si ya existen, Kafka responde sin error fatal.
func EnsureTopics(ctx context.Context, brokers []string, topics ...string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("sin brokers configurados")
	}
	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("dial kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("controller: %w", err)
	}
	ctrlConn, err := kafka.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("dial controller: %w", err)
	}
	defer ctrlConn.Close()

	cfgs := make([]kafka.TopicConfig, 0, len(topics))
	for _, t := range topics {
		cfgs = append(cfgs, kafka.TopicConfig{
			Topic:             t,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}
	if err := ctrlConn.CreateTopics(cfgs...); err != nil {
		return fmt.Errorf("create topics: %w", err)
	}
	return nil
}

// Publisher publica mensajes en Kafka. El topic se indica por mensaje, de modo
// que un mismo Publisher sirve para el topic principal y la DLQ.
type Publisher struct {
	w *kafka.Writer
}

func NewPublisher(brokers []string) *Publisher {
	return &Publisher{
		w: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Balancer:               &kafka.LeastBytes{},
			RequiredAcks:           kafka.RequireAll, // durabilidad: el dinero no se pierde
			AllowAutoTopicCreation: false,
			WriteTimeout:           10 * time.Second,
		},
	}
}

// Publish envía un mensaje con key (usamos cuota_id para ordenar por cuota).
func (p *Publisher) Publish(ctx context.Context, topic string, key, value []byte) error {
	return p.w.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

func (p *Publisher) Close() error { return p.w.Close() }

// NewReader crea un reader con consumer group para consumo confiable
// (commit manual de offset tras procesar).
func NewReader(brokers []string, groupID, topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0, // commit manual
	})
}
