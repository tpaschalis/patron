// +build integration

package amqp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/beatlabs/patron/encoding/json"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

func TestConsumeAndPublish(t *testing.T) {
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

	ch := setupRabbitMQPublisher(t)

	// Wait for everything to be set up properly.
	time.Sleep(5000 * time.Millisecond)

	type args struct {
		body string
		ct   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"success", args{`{"broker":"🐰"}`, json.Type}, false},
		{"failure - invalid content-type", args{`amqp rocks!`, "text/plain"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendRabbitMQMessage(t, ch, tt.args.body, tt.args.ct)
			if tt.wantErr == false {
				assert.Len(t, msgChan, 1)
			} else {
				assert.Len(t, errChan, 1)
			}
		})
	}
}

func TestConsumeFailures(t *testing.T) {

	type args struct {
		url   string
		queue string
		ex    Exchange
	}
	tests := []struct {
		name    string
		args    args
		wantErr string
	}{
		{"failure due to url",
			args{
				url:   "foo",
				queue: "queue",
				ex:    *validExch,
			},
			"failed initialize consumer: failed to dial @ foo: AMQP scheme must be either 'amqp://' or 'amqps://'",
		},
		{
			"failure due to exchange",
			args{
				url:   "amqp://guest:guest@localhost/",
				queue: "queue",
				ex:    Exchange{"foo", "bar"},
			},
			"failed initialize consumer: failed to declare exchange: Exception (503) Reason: \"COMMAND_INVALID - invalid exchange type 'bar'\"",
		},
		{"failure due to queue newline",
			args{
				url:   "amqp://guest:guest@localhost/",
				queue: "\n",
				ex:    *validExch,
			},
			"failed initialize consumer: failed initialize consumer: Exception (404) Reason: \"NOT_FOUND - no queue '\\n' in vhost '/'\"",
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			f, _ := New(tt.args.url, tt.args.queue, tt.args.ex)
			c, _ := f.Create()
			msgChan, errChan, err := c.Consume(ctx)

			assert.Nil(t, msgChan)
			assert.Nil(t, errChan)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestConsumeAndCancel(t *testing.T) {
	f := &Factory{
		url:      "amqp://guest:guest@localhost/",
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

func TestConsumeAndClose(t *testing.T) {
	f := &Factory{
		url:      "amqp://guest:guest@localhost/",
		queue:    "queue",
		exchange: *validExch,
		bindings: []string{},
	}
	c, err := f.Create()
	require.NoError(t, err)
	ctx := context.Background()

	_, _, err = c.Consume(ctx)
	assert.NoError(t, err)
	err = c.Close()
	assert.NoError(t, err)
}

// Small default publisher for testing purposes.
func setupRabbitMQPublisher(t *testing.T) *amqp.Channel {

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	require.NoErrorf(t, err, "failed to connect to RabbitMQ consumer: %v", err)

	ch, err := conn.Channel()
	require.NoErrorf(t, err, "failed to open a connection channel: %v", err)

	_, err = ch.QueueDeclare(
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
	require.NoErrorf(t, err, "failed to bind queue: %v", err)

	return ch
}

func sendRabbitMQMessage(t *testing.T, ch *amqp.Channel, body, ct string) {
	err := ch.Publish(validExch.name, "queue", false, false, amqp.Publishing{
		ContentType: ct,
		Body:        []byte(body),
	})
	require.NoErrorf(t, err, "failed to publish message: %v", err)
	time.Sleep(1000 * time.Millisecond)
}
