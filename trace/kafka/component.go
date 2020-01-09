package kafka

import (
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/opentracing/opentracing-go"
)

// RequiredAcks is used in Produce Requests to tell the broker how many replica acknowledgements
// it must see before responding.
type RequiredAcks int16

const (
	// NoResponse doesn't send any response, the TCP ACK is all you get.
	NoResponse RequiredAcks = 0
	// WaitForLocal waits for only the local commit to succeed before responding.
	WaitForLocal RequiredAcks = 1
	// WaitForAll waits for all in-sync replicas to commit before responding.
	WaitForAll RequiredAcks = -1
)

const fieldSetMsg = "Setting property '%v' for '%v'"

// Builder gathers all required and optional properties, in order
// to construct a Kafka AsyncProducer.
type Builder struct {
	brokers     []string
	cfg         *sarama.Config
	chErr       chan error
	tag         opentracing.Tag
	enc         encoding.EncodeFunc
	contentType string
	errors      []error
}

// NewBuilder initiates the AsyncProducer builder chain.
// The builder instantiates the component using default values for
// EncodeFunc and Content-Type header.
func NewBuilder() *Builder {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0
	var errs []error
	return &Builder{
		cfg:         cfg,
		chErr:       make(chan error),
		tag:         opentracing.Tag{Key: "type", Value: "async"},
		enc:         json.Encode,
		contentType: json.Type,
		errors:      errs,
	}
}

// WithTimeout sets the dial timeout for the AsyncProducer.
func (ab *Builder) WithTimeout(dial time.Duration) *Builder {
	if dial <= 0*time.Second {
		ab.errors = append(ab.errors, errors.New("dial timeout has to be positive"))
	}
	ab.cfg.Net.DialTimeout = dial
	log.Info(fieldSetMsg, "dial timeout", dial)

	return ab
}

// WithVersion sets the kafka versionfor the AsyncProducer.
func (ab *Builder) WithVersion(version string) *Builder {
	if version == "" {
		ab.errors = append(ab.errors, errors.New("version is required"))
		return ab
	}
	v, err := sarama.ParseKafkaVersion(version)
	if err != nil {
		ab.errors = append(ab.errors, errors.New("failed to parse kafka version"))
		return ab
	}
	log.Info(fieldSetMsg, "version", version)
	ab.cfg.Version = v

	return ab
}

// WithRequiredAcksPolicy adjusts how many replica acknowledgements
// broker must see before responding
func (ab *Builder) WithRequiredAcksPolicy(ack RequiredAcks) *Builder {
	log.Info(fieldSetMsg, "required acks", ack)
	ab.cfg.Producer.RequiredAcks = sarama.RequiredAcks(ack)

	return ab
}

// WithEncoder sets a specific encoder implementation and Content-Type string header;
// if no option is provided it defaults to json.
func (ab *Builder) WithEncoder(enc encoding.EncodeFunc, contentType string) *Builder {
	if enc == nil {
		ab.errors = append(ab.errors, errors.New("encoder is nil"))
	} else {
		log.Info(fieldSetMsg, "encoder", enc)
		ab.enc = enc
	}
	if contentType == "" {
		ab.errors = append(ab.errors, errors.New("content type is empty"))
	} else {
		log.Info(fieldSetMsg, "content type", contentType)
		ab.contentType = contentType
	}

	return ab
}

// WithBrokers sets the list of brokers the AsyncProducer will work with.
func (ab *Builder) WithBrokers(brokers []string) *Builder {
	if len(brokers) == 0 {
		ab.errors = append(ab.errors, errors.New("brokers list is empty"))
	} else {
		log.Info(fieldSetMsg, "brokers", brokers)
		ab.brokers = append(ab.brokers, brokers...)
	}

	return ab
}

// Create constructs the AsyncProducer component by applying the gathered properties.
func (ab *Builder) Create() (*AsyncProducer, error) {
	if len(ab.errors) > 0 {
		return nil, errors.Aggregate(ab.errors...)
	}

	prod, err := sarama.NewAsyncProducer(ab.brokers, ab.cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create async producer")
	}

	ap := AsyncProducer{
		cfg:         ab.cfg,
		prod:        prod,
		chErr:       ab.chErr,
		enc:         ab.enc,
		contentType: ab.contentType,
		tag:         ab.tag,
	}

	go ap.propagateError()
	return &ap, nil
}
