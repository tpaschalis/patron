package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	producerComponent = "kafka-async-producer"
)

var countMessagesSent *prometheus.CounterVec
var countMessageSendErrors *prometheus.CounterVec
var countMessageCreationErrors *prometheus.CounterVec

// CountMessagesSentInc increments the countMessagesSent counter.
func CountMessagesSentInc(topic string) {
	countMessagesSent.WithLabelValues(topic).Inc()
}

// CountMessageSendErrorsInc increments the countMessageSendErrors counter.
func CountMessageSendErrorsInc(topic string) {
	countMessageSendErrors.WithLabelValues(topic).Inc()
}

// CountMessageCreationErrorsInc increments the countMessageCreationErrors counter.
func CountMessageCreationErrorsInc(topic string) {
	countMessageCreationErrors.WithLabelValues(topic).Inc()
}

func init() {
	countMessagesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "kafka_async_producer",
			Name:      "messages_sent",
			Help:      "Messages sent counter, classified by topic",
		}, []string{"topic"},
	)
	countMessageSendErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "kafka_async_producer",
			Name:      "message_send_errors",
			Help:      "Message send errors counter, classified by topic",
		}, []string{"topic"},
	)
	countMessageCreationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "kafka_async_producer",
			Name:      "message_creation_errors",
			Help:      "Message creation errors counter, classified by topic",
		}, []string{"topic"},
	)
	prometheus.MustRegister(
		countMessagesSent,
		countMessageSendErrors,
		countMessageCreationErrors,
	)
}

// Message abstraction of a Kafka message.
type Message struct {
	topic string
	body  interface{}
	key   *string
}

// NewMessage creates a new message.
func NewMessage(t string, b interface{}) *Message {
	return &Message{topic: t, body: b}
}

// NewMessageWithKey creates a new message with an associated key.
func NewMessageWithKey(t string, b interface{}, k string) (*Message, error) {
	if k == "" {
		return nil, errors.New("key string can not be null")
	}
	return &Message{topic: t, body: b, key: &k}, nil
}

// Producer interface for Kafka.
type Producer interface {
	Send(ctx context.Context, msg *Message) error
	Error() <-chan error
	Close() error
}

// AsyncProducer defines a async Kafka producer.
type AsyncProducer struct {
	cfg         *sarama.Config
	prod        sarama.AsyncProducer
	chErr       chan error
	tag         opentracing.Tag
	enc         encoding.EncodeFunc
	contentType string
}

// Send a message to a topic.
func (ap *AsyncProducer) Send(ctx context.Context, msg *Message) error {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(producerComponent, msg.topic),
		producerComponent, ext.SpanKindProducer, ap.tag,
		opentracing.Tag{Key: "topic", Value: msg.topic})
	pm, err := ap.createProducerMessage(ctx, msg, sp)
	if err != nil {
		CountMessageCreationErrorsInc(msg.topic)
		trace.SpanError(sp)
		return err
	}
	CountMessagesSentInc(msg.topic)
	ap.prod.Input() <- pm
	trace.SpanSuccess(sp)
	return nil
}

// Error returns a chanel to monitor for errors.
func (ap *AsyncProducer) Error() <-chan error {
	return ap.chErr
}

// Close gracefully the producer.
func (ap *AsyncProducer) Close() error {
	err := ap.prod.Close()
	if err != nil {
		return fmt.Errorf("failed to close sync producer: %w", err)
	}
	return nil
}

func (ap *AsyncProducer) propagateError() {
	for pe := range ap.prod.Errors() {
		CountMessageSendErrorsInc(pe.Msg.Topic)
		ap.chErr <- fmt.Errorf("failed to send message: %w", pe)
	}
}

func (ap *AsyncProducer) createProducerMessage(ctx context.Context, msg *Message, sp opentracing.Span) (*sarama.ProducerMessage, error) {
	c := kafkaHeadersCarrier{}
	err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tracing headers: %w", err)
	}
	c.Set(encoding.ContentTypeHeader, ap.contentType)

	var saramaKey sarama.Encoder
	if msg.key != nil {
		saramaKey = sarama.StringEncoder(*msg.key)
	}

	b, err := ap.enc(msg.body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message body")
	}

	c.Set(correlation.HeaderID, correlation.IDFromContext(ctx))
	return &sarama.ProducerMessage{
		Topic:   msg.topic,
		Key:     saramaKey,
		Value:   sarama.ByteEncoder(b),
		Headers: c,
	}, nil
}

type kafkaHeadersCarrier []sarama.RecordHeader

// Set implements Set() of opentracing.TextMapWriter.
func (c *kafkaHeadersCarrier) Set(key, val string) {
	*c = append(*c, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}
