// +build integration

package amqp

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	amqpClient "github.com/beatlabs/patron/client/amqp"
	amqpConsumer "github.com/beatlabs/patron/component/async/amqp"

	"github.com/beatlabs/patron/encoding/json"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	runtime       *amqpRuntime
	validExch, _  = amqpConsumer.NewExchange("e", amqp.ExchangeDirect)
	validExchName = "e"
)

func TestMain(m *testing.M) {
	var err error
	runtime, err = create(60 * time.Second)
	if err != nil {
		fmt.Printf("could not create mysql runtime: %v\n", err)
		os.Exit(1)
	}
	defer func() {

	}()
	exitCode := m.Run()

	ee := runtime.Teardown()
	if len(ee) > 0 {
		for _, err = range ee {
			fmt.Printf("could not tear down containers: %v\n", err)
		}
	}
	os.Exit(exitCode)
}

func TestPublisherSuccess(t *testing.T) {
	ctx := context.Background()
	pub, err := amqpClient.NewPublisher("amqp://guest:guest@localhost:5672/", "exchangeName")
	assert.NoError(t, err)

	originalMsg, err := amqpClient.NewJSONMessage(`{"status":"received"}`)
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
			_, err := amqpClient.NewPublisher(tt.args.url, tt.args.exchangeName)
			assert.EqualError(t, err, tt.publisherErr)

		})
	}
}

func TestPublishIntoClosedChannel(t *testing.T) {
	ctx := context.Background()
	pub, err := amqpClient.NewPublisher("amqp://guest:guest@localhost:5672/", "foo")
	assert.NoError(t, err)
	msg, err := amqpClient.NewJSONMessage(`"foo": "bar"`)
	assert.NoError(t, err)

	err = pub.Close(ctx)
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

func TestConsumeAndPublish(t *testing.T) {

	// Setup consumer.
	f, err := amqpConsumer.New(
		"amqp://guest:guest@localhost/",
		"async-amqp-queue",
		*validExch,
	)
	require.NoErrorf(t, err, "failed to create factory: %v", err)

	c, err := f.Create()
	require.NoErrorf(t, err, "failed to create consumer: %v", err)
	ctx := context.Background()
	msgChan, errChan, err := c.Consume(ctx)
	assert.NotNil(t, msgChan)
	assert.NotNil(t, errChan)
	assert.NoError(t, err)

	conn, ch := setupRabbitMQPublisher(t)
	defer func() {
		err := conn.Close()
		assert.NoError(t, err)
	}()

	//Wait for everything to be set up properly.
	time.Sleep(2000 * time.Millisecond)

	type args struct {
		body string
		ct   string
	}
	tests := map[string]struct {
		args    args
		wantErr bool
	}{
		"success":                        {args{`{"broker":"🐰"}`, json.Type}, false},
		"failure - invalid content-type": {args{`amqp rocks!`, "text/plain"}, true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sendRabbitMQMessage(t, ch, tt.args.body, tt.args.ct)
			if tt.wantErr == false {
				msg := <-msgChan
				assert.NotNil(t, msg)
			} else {
				msg := <-errChan
				assert.NotNil(t, msg)
			}
		})
	}
}

func TestConsumeFailures(t *testing.T) {

	type args struct {
		url   string
		queue string
		ex    amqpConsumer.Exchange
	}
	tests := map[string]struct {
		args    args
		wantErr string
	}{
		"failure due to url": {
			args: args{
				url:   "foo",
				queue: "async-amqp-queue",
				ex:    *validExch,
			},
			wantErr: "failed initialize consumer: failed to dial @ foo: AMQP scheme must be either 'amqp://' or 'amqps://'",
		},
		"failure due to queue newline": {
			args: args{
				url:   "amqp://guest:guest@localhost/",
				queue: "\n",
				ex:    *validExch,
			},
			wantErr: "failed initialize consumer: failed initialize consumer: Exception (404) Reason: \"NOT_FOUND - no queue '\\n' in vhost '/'\"",
		},
	}

	ctx := context.Background()

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			f, _ := amqpConsumer.New(tt.args.url, tt.args.queue, tt.args.ex)
			c, _ := f.Create()
			msgChan, errChan, err := c.Consume(ctx)

			assert.Nil(t, msgChan)
			assert.Nil(t, errChan)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestConsumeAndCancel(t *testing.T) {
	var validExch, _ = amqpConsumer.NewExchange("e", amqp.ExchangeDirect)

	f, err := amqpConsumer.New(
		"amqp://guest:guest@localhost/",
		"async-amqp-queue",
		*validExch,
	)
	require.NoErrorf(t, err, "failed to create factory: %v", err)

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
	f, err := amqpConsumer.New(
		"amqp://guest:guest@localhost/",
		"async-amqp-queue",
		*validExch,
	)
	require.NoErrorf(t, err, "failed to create factory: %v", err)

	c, err := f.Create()
	require.NoError(t, err)
	ctx := context.Background()

	_, _, err = c.Consume(ctx)
	assert.NoError(t, err)
	err = c.Close()
	assert.NoError(t, err)
}

// Small default publisher for testing purposes.
func setupRabbitMQPublisher(t *testing.T) (*amqp.Connection, *amqp.Channel) {

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	require.NoErrorf(t, err, "failed to connect to RabbitMQ consumer: %v", err)

	ch, err := conn.Channel()
	require.NoErrorf(t, err, "failed to open a connection channel: %v", err)

	_, err = ch.QueueDeclare(
		"async-amqp-queue", // name
		true,               // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	require.NoErrorf(t, err, "failed to declare a queue: %v", err)

	err = ch.QueueBind(
		"async-amqp-queue", // queue name
		"async-amqp-queue", // routing key
		validExchName,      // exchange
		false,
		nil,
	)
	require.NoErrorf(t, err, "failed to bind queue: %v", err)

	return conn, ch
}

func sendRabbitMQMessage(t *testing.T, ch *amqp.Channel, body, ct string) {
	err := ch.Publish(validExchName, "async-amqp-queue", false, false, amqp.Publishing{
		ContentType: ct,
		Body:        []byte(body),
	})
	require.NoErrorf(t, err, "failed to publish message: %v", err)
	time.Sleep(3000 * time.Millisecond) // throttle messages to avoid queue saturation
}
