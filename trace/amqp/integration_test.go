// !build integration

package amqp

import (
	"context"
	"testing"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestPublisherSuccess(t *testing.T) {
	ctx := context.Background()
	pub, err := NewPublisher("amqp://guest:guest@localhost:5672/", "exchangeName")
	assert.NoError(t, err)

	originalMsg, err := NewJSONMessage(`{"status":"received"}`)
	assert.NoError(t, err)
	expectedMsg := `"{\"status\":\"received\"}"`

	msgs := setupRabbitMQConsumer(t)
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

			args{"", ""},
			"url is required",
		},
		"missing exchange": {

			args{"foo", ""},
			"exchange is required",
		},
		"incorrect URL": {

			args{"foo", "bar"},
			"failed to open RabbitMq connection: AMQP scheme must be either 'amqp://' or 'amqps://'",
		},
		"incorrect exchange": {

			args{"amqp://guest:guest@localhost:5672/", "\n"},
			"failed to declare exchange: Exception (403) Reason: \"ACCESS_REFUSED - operation not permitted on the default exchange\"",
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

func setupRabbitMQConsumer(t *testing.T) <-chan amqp.Delivery {
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

	return msgs
}
