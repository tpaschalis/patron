// +build integration

package amqp

import (
	"context"
	"testing"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublisherSuccess(t *testing.T) {
	ctx := context.Background()
	pub, err := NewPublisher("amqp://guest:guest@localhost:5672/", "exchangeName")
	assert.NoError(t, err)

	originalMsg, err := NewJSONMessage(`{"status":"received"}`)
	assert.NoError(t, err)
	expectedMsg := `"{\"status\":\"received\"}"`

	conn, msgs := setupRabbitMQConsumer(t)
	defer func() {
		err := conn.Close()
		assert.NoError(t, err)
	}()
	err = pub.Publish(ctx, originalMsg)
	assert.NoError(t, err)

	received := <-msgs
	assert.Equal(t, expectedMsg, string(received.Body))

	err = pub.Close(ctx)
	assert.NoError(t, err)
}

func TestPublisherFailures(t *testing.T) {
	type args struct {
		url          string
		exchangeName string
	}
	tests := map[string]struct {
		args         args
		publisherErr string
	}{
		"missing url": {
			args:         args{"", ""},
			publisherErr: "url is required",
		},
		"missing exchange": {
			args:         args{"foo", ""},
			publisherErr: "exchange is required",
		},
		"incorrect URL": {
			args:         args{"foo", "bar"},
			publisherErr: "failed to open RabbitMq connection: AMQP scheme must be either 'amqp://' or 'amqps://'",
		},
		"incorrect exchange": {
			args:         args{"amqp://guest:guest@localhost:5672/", "\n"},
			publisherErr: "failed to declare exchange: Exception (403) Reason: \"ACCESS_REFUSED - operation not permitted on the default exchange\"",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewPublisher(tt.args.url, tt.args.exchangeName)
			assert.EqualError(t, err, tt.publisherErr)

		})
	}
}

func TestPublishIntoClosedChannel(t *testing.T) {
	ctx := context.Background()
	pub, err := NewPublisher("amqp://guest:guest@localhost:5672/", "foo")
	assert.NoError(t, err)
	msg, err := NewJSONMessage(`"foo": "bar"`)
	assert.NoError(t, err)

	err = pub.ch.Close()
	assert.NoError(t, err)
	err = pub.Publish(ctx, msg)
	assert.EqualError(t, err, "failed to publish message: Exception (504) Reason: \"channel/connection is not open\"")
}

func setupRabbitMQConsumer(t *testing.T) (*amqp.Connection, <-chan amqp.Delivery) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost/")
	require.NoError(t, err)

	ch, err := conn.Channel()
	require.NoError(t, err)

	err = ch.ExchangeDeclare(
		"exchangeName",      // name
		amqp.ExchangeFanout, // kind
		true,                // durable
		false,               // autoDelete
		false,               // internal
		false,               // noWait
		nil,                 // args
	)
	require.NoError(t, err)

	q, err := ch.QueueDeclare(
		"trace-amqp-queue", // name
		true,               // durable
		false,              // audoDelete
		false,              // exclusive
		false,              // noWait
		nil,                // args
	)
	require.NoError(t, err)

	err = ch.QueueBind(
		q.Name,         // name
		"",             // key
		"exchangeName", // exchange
		false,          // noWait
		nil,            // args
	)
	require.NoError(t, err)

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // autoAck
		false,  // exclusive
		false,  // noLocal
		false,  // noWait
		nil,    // args
	)
	require.NoError(t, err)

	return conn, msgs
}