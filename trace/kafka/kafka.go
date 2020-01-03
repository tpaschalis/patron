package kafka

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
)

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

// NewJSONMessage creates a new message with a JSON encoded body.
func NewJSONMessage(t string, d interface{}) (*Message, error) {
	b, err := json.Encode(d)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON encode: %w", err)
	}
	return &Message{topic: t, body: b}, nil
}

// NewJSONMessageWithKey creates a new message with a JSON encoded body and a message key.
func NewJSONMessageWithKey(t string, d interface{}, k string) (*Message, error) {
	if k == "" {
		return nil, errors.New("key string can not be null")
	}
	b, err := json.Encode(d)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON encode: %w", err)
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
	cfg   *sarama.Config
	prod  sarama.AsyncProducer
	chErr chan error
	tag   opentracing.Tag
	enc   encoding.EncodeFunc
	ct    string
}

// NewAsyncProducer creates a new async producer with default configuration.
func NewAsyncProducer(brokers []string, oo ...OptionFunc) (*AsyncProducer, error) {

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0

	ap := AsyncProducer{cfg: cfg, chErr: make(chan error), tag: opentracing.Tag{Key: "type", Value: "async"}, enc: json.Encode}

	for _, o := range oo {
		err := o(&ap)
		if err != nil {
			return nil, err
		}
	}

	prod, err := sarama.NewAsyncProducer(brokers, ap.cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create async producer")
	}
	ap.prod = prod
	go ap.propagateError()
	return &ap, nil
}

// Send a message to a topic.
func (ap *AsyncProducer) Send(ctx context.Context, msg *Message) error {

	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(trace.KafkaAsyncProducerComponent, msg.topic),
		trace.KafkaAsyncProducerComponent, ext.SpanKindProducer, ap.tag,
		opentracing.Tag{Key: "topic", Value: msg.topic})
	pm, err := createProducerMessage(ctx, msg, sp, ap.enc, ap.ct)
	if err != nil {
		trace.SpanError(sp)
		return err
	}
	ap.prod.Input() <- pm
	trace.SpanSuccess(sp)
	return nil
}

// SendRaw sends an already-serialized message to a topic.
func (ap *AsyncProducer) SendRaw(ctx context.Context, msg *Message, ct string) error {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(trace.KafkaAsyncProducerComponent, msg.topic),
		trace.KafkaAsyncProducerComponent, ext.SpanKindProducer, ap.tag,
		opentracing.Tag{Key: "topic", Value: msg.topic})
	pm, err := createProducerMessage(ctx, msg, sp, rawEncodeFunc, ct)
	if err != nil {
		trace.SpanError(sp)
		return err
	}
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
		ap.chErr <- fmt.Errorf("failed to send message: %w", pe)
	}
}

func createProducerMessage(ctx context.Context, msg *Message, sp opentracing.Span, enc encoding.EncodeFunc, ct string) (*sarama.ProducerMessage, error) {
	c := kafkaHeadersCarrier{}
	err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tracing headers: %w", err)
	}

	if len(ct) != 0 {
		c.Set(encoding.ContentTypeHeader, ct)
	}

	var saramaKey, saramaBody sarama.Encoder
	if msg.key != nil {
		k, err := rawEncodeFunc(*msg.key)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode partition key")
		}
		saramaKey = sarama.ByteEncoder(k)
	}

	b, err := enc(msg.body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode message body")
	}
	saramaBody = sarama.ByteEncoder(b)

	c.Set(correlation.HeaderID, correlation.IDFromContext(ctx))
	return &sarama.ProducerMessage{
		Topic:   msg.topic,
		Key:     saramaKey,
		Value:   saramaBody,
		Headers: c,
	}, nil
}

type kafkaHeadersCarrier []sarama.RecordHeader

// Set implements Set() of opentracing.TextMapWriter.
func (c *kafkaHeadersCarrier) Set(key, val string) {
	*c = append(*c, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}

func rawEncodeFunc(v interface{}) ([]byte, error) {
	b, ok := v.([]byte)
	if ok {
		return b, nil
	}
	s, ok := v.(string)
	if ok {
		return []byte(s), nil
	}
	return nil, errors.New("could not encode msg with default encodefunc")
}
