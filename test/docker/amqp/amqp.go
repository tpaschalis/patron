package amqp

import (
	"fmt"
	"time"

	patronDocker "github.com/beatlabs/patron/test/docker"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
)

type amqpRuntime struct {
	patronDocker.Runtime
}

// Create initializes a RabbitMQ docker runtime.
func create(expiration time.Duration) (*amqpRuntime, error) {
	br, err := patronDocker.NewRuntime(expiration)
	if err != nil {
		return nil, fmt.Errorf("could not create base runtime: %w", err)
	}

	runtime := &amqpRuntime{Runtime: *br}

	runOptions := &dockertest.RunOptions{Repository: "rabbitmq",
		Tag: "5.7.25",
		PortBindings: map[docker.Port][]docker.PortBinding{
			"15672/tcp": {{HostIP: "", HostPort: ""}},
			"5672/tcp":  {{HostIP: "", HostPort: ""}},
		},
	}
	_, err = runtime.RunWithOptions(runOptions)
	if err != nil {
		return nil, fmt.Errorf("could not start rabbitmq: %w", err)
	}

	// wait until the container is ready
	time.Sleep(30 * time.Second)

	return runtime, nil
}
