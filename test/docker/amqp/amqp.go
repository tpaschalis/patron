package amqp

import (
	"fmt"
	"time"

	"github.com/ory/dockertest/docker"

	patronDocker "github.com/beatlabs/patron/test/docker"
	// Integration test.
	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest"
)

type amqpRuntime struct {
	patronDocker.Runtime
}

// Create initializes a Sql docker runtime.
func create(expiration time.Duration) (*amqpRuntime, error) {
	br, err := patronDocker.NewRuntime(expiration)
	if err != nil {
		return nil, fmt.Errorf("could not create base runtime: %w", err)
	}

	runtime := &amqpRuntime{Runtime: *br}

	runOptions := &dockertest.RunOptions{Repository: "mysql",
		Tag: "5.7.25",
		PortBindings: map[docker.Port][]docker.PortBinding{
			"15672/tcp": {{HostIP: "", HostPort: ""}},
			"5672/tcp":  {{HostIP: "", HostPort: ""}},
		},
		//ExposedPorts: []string{"3306/tcp", "33060/tcp"},
	}
	_, err = runtime.RunWithOptions(runOptions)
	if err != nil {
		return nil, fmt.Errorf("could not start mysql: %w", err)
	}

	// wait until the container is ready
	err = runtime.Pool().Retry(func() error {
		time.Sleep(60 * time.Second)
		return nil
		// db, err := sql.Open("mysql", runtime.DSN())
		// if err != nil {
		// 	// container not ready ... return error to try again
		// 	return err
		// }
		// return db.Ping()
	})
	if err != nil {
		for _, err1 := range runtime.Teardown() {
			fmt.Printf("failed to teardown: %v\n", err1)
		}
		return nil, fmt.Errorf("container not ready: %w", err)
	}

	return runtime, nil
}
