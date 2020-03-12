// !build integration

package amqp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublisherSuccess(t *testing.T) {
	ctx := context.Background()
	pub, err := NewPublisher("amqp://guest:guest@localhost:5672/", "exchangeName")
	assert.NoError(t, err)

	msg, err := NewJSONMessage(`{"broker":"🐰"}`)
	assert.NoError(t, err)

	err = pub.Publish(ctx, msg)
	assert.NoError(t, err)

	err = pub.Close(ctx)
	assert.NoError(t, err)
}

func TestPublisherFailures(t *testing.T) {
	type args struct {
		url          string
		exchangeName string
	}
	tests := []struct {
		name         string
		args         args
		publisherErr string
	}{
		{
			"missing url",
			args{"", ""},
			"url is required",
		},
		{
			"missing exchange",
			args{"foo", ""},
			"exchange is required",
		},
		{
			"incorrect URL",
			args{"foo", "bar"},
			"failed to open RabbitMq connection: AMQP scheme must be either 'amqp://' or 'amqps://'",
		},
		{
			"incorrect exchange",
			args{"amqp://guest:guest@localhost:5672/", "\n"},
			"failed to declare exchange: Exception (403) Reason: \"ACCESS_REFUSED - operation not permitted on the default exchange\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPublisher(tt.args.url, tt.args.exchangeName)
			assert.EqualError(t, err, tt.publisherErr)

		})
	}
}

func TestPublishIntoClosedChannel(t *testing.T) {
	ctx := context.Background()
	pub, _ := NewPublisher("amqp://guest:guest@localhost:5672/", "foo")
	msg, _ := NewJSONMessage(`"foo": "bar"`)

	err := pub.ch.Close()
	assert.NoError(t, err)
	err = pub.Publish(ctx, msg)
	assert.EqualError(t, err, "failed to publish message: Exception (504) Reason: \"channel/connection is not open\"")
}
