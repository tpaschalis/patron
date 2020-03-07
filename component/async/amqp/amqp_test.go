package amqp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

var validExch, _ = NewExchange("e", amqp.ExchangeDirect)

func Test_message(t *testing.T) {
	b, err := json.Encode("test")
	assert.NoError(t, err)
	del := &amqp.Delivery{
		Body: b,
	}
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	sp := opentracing.StartSpan("test")
	ctx := context.Background()
	m := message{
		ctx:    ctx,
		del:    del,
		dec:    json.DecodeRaw,
		span:   sp,
		source: "thequeue",
	}
	assert.Equal(t, ctx, m.Context())
	var data string
	assert.NoError(t, m.Decode(&data))
	assert.Equal(t, "test", data)
	assert.Error(t, m.Ack())
	assert.Error(t, m.Nack())
	assert.Equal(t, "thequeue", m.Source())
	assert.Equal(t, []byte(`"test"`), m.Payload())
}

func TestNewExchange(t *testing.T) {
	type args struct {
		name string
		kind string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"success, kind fanout", args{name: "abc", kind: amqp.ExchangeFanout}, false},
		{"success, kind headers", args{name: "abc", kind: amqp.ExchangeHeaders}, false},
		{"success, kind topic", args{name: "abc", kind: amqp.ExchangeTopic}, false},
		{"success, kind direct", args{name: "abc", kind: amqp.ExchangeDirect}, false},
		{"fail, empty name", args{name: "", kind: amqp.ExchangeTopic}, true},
		{"fail, empty kind", args{name: "abc", kind: ""}, true},
		{"fail, invalid kind", args{name: "abc", kind: "def"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exc, err := NewExchange(tt.args.name, tt.args.kind)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, exc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, exc)
			}
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		url      string
		queue    string
		exchange Exchange
		opt      OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"success", args{url: "amqp://guest:guest@localhost:5672/", queue: "q", exchange: *validExch, opt: Buffer(100)}, false},
		{"fail, invalid url", args{url: "", queue: "q", exchange: *validExch, opt: Buffer(100)}, true},
		{"fail, invalid queue name", args{url: "url", queue: "", exchange: *validExch, opt: Buffer(100)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.url, tt.args.queue, tt.args.exchange, tt.args.opt)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestFactory_Create(t *testing.T) {
	type fields struct {
		oo []OptionFunc
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "success", wantErr: false},
		{name: "invalid option", fields: fields{oo: []OptionFunc{Buffer(-10)}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{
				url:      "url",
				queue:    "queue",
				exchange: *validExch,
				bindings: []string{},
				oo:       tt.fields.oo,
			}
			got, err := f.Create()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_mapHeader(t *testing.T) {
	hh := amqp.Table{"test1": 10, "test2": 0.11}
	mm := map[string]string{"test1": "10", "test2": "0.11"}
	assert.Equal(t, mm, mapHeader(hh))
}

func Test_getCorrelationID(t *testing.T) {
	withID := amqp.Table{correlation.HeaderID: "123"}
	withoutID := amqp.Table{correlation.HeaderID: ""}
	missingHeader := amqp.Table{}
	type args struct {
		hh amqp.Table
	}
	tests := map[string]struct {
		args args
	}{
		"with id":        {args: args{hh: withID}},
		"without id":     {args: args{hh: withoutID}},
		"missing header": {args: args{hh: missingHeader}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, getCorrelationID(tt.args.hh))
		})
	}
}

func TestConsumeAndCancel(t *testing.T) {
	f := &Factory{
		url:      "amqp://",
		queue:    "queue",
		exchange: *validExch,
		bindings: []string{},
	}
	c, err := f.Create()
	require.NoError(t, err)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	msgChan, errChan, err := c.Consume(ctx)
	cancel()
	assert.Empty(t, msgChan)
	assert.Empty(t, errChan)
	assert.NoError(t, err)
}

func TestConsumeAndDeliver(t *testing.T) {

	// Setup consumer.
	f := &Factory{
		url:      "amqp://guest:guest@localhost/",
		queue:    "queue",
		exchange: *validExch,
		bindings: []string{},
	}
	c, err := f.Create()
	require.NoErrorf(t, err, "failed to create consumer: %v", err)
	ctx := context.Background()
	msgChan, errChan, err := c.Consume(ctx)
	assert.NotNil(t, msgChan)
	assert.NotNil(t, errChan)
	assert.NoError(t, err)

	type args struct {
		body string
		ct   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"failure - invalid content-type", args{`amqp`, "text/plain"}, true},
		{"success", args{`{"broker":"🐰"}`, json.Type}, false},
	}

	// Wait for everything to be set up properly.
	time.Sleep(500 * time.Millisecond)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendRabbitMQMessage(t, tt.args.body, tt.args.ct)
			if tt.wantErr == false {
				assert.NotEmpty(t, msgChan)
			} else {
				assert.NotEmpty(t, errChan)
			}
		})
	}
}

func sendRabbitMQMessage(t *testing.T, body, ct string) {
	// Build small publisher.
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	require.NoErrorf(t, err, "failed to connect to RabbitMQ consumer: %v", err)
	defer func() {
		err = conn.Close()
		assert.NoError(t, err)
	}()

	ch, err := conn.Channel()
	require.NoErrorf(t, err, "failed to open a connection channel: %v", err)
	defer func() {
		err = ch.Close()
		assert.NoError(t, err)
	}()

	q, err := ch.QueueDeclare(
		"queue", // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	require.NoErrorf(t, err, "failed to declare a queue: %v", err)

	err = ch.QueueBind(
		"queue",        // queue name
		"queue",        // routing key
		validExch.name, // exchange
		false,
		nil,
	)

	// Send message.
	err = ch.Publish(validExch.name, q.Name, false, false, amqp.Publishing{
		ContentType: ct,
		Body:        []byte(body),
	})
	require.NoErrorf(t, err, "failed to publish message: %v", err)
}
