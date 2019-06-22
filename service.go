package patron

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
)

// Component interface for implementing service components.
type Component interface {
	Run(ctx context.Context) error
	Info() map[string]interface{}
}

// Service is responsible for managing and setting up everything.
// The service will start by default a HTTP component in order to host management endpoint.
type service struct {
	cps           []Component
	termSig       chan os.Signal
	sighupHandler func()
}

func new(components []Component, sighubHandler func()) (*service, error) {

	if len(components) == 0 {
		return nil, errors.New("no components provided")
	}

	if sighubHandler == nil {
		sighubHandler = func() { log.Info("SIGHUP received: nothing setup") }
	}

	s := service{
		cps:           components,
		termSig:       make(chan os.Signal, 1),
		sighupHandler: sighubHandler,
	}

	s.setupOSSignal()
	return &s, nil
}

func (s *service) setupOSSignal() {
	signal.Notify(s.termSig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
}

// Run starts up all service components and monitors for errors.
// If a component returns a error the service is responsible for shutting down
// all components and terminate itself.
func (s *service) Run() error {
	ctx, cnl := context.WithCancel(context.Background())
	chErr := make(chan error, len(s.cps))
	wg := sync.WaitGroup{}
	wg.Add(len(s.cps))
	for _, cp := range s.cps {
		go func(c Component) {
			defer wg.Done()
			chErr <- c.Run(ctx)
		}(cp)
	}

	var ee []error
	ee = append(ee, s.waitTermination(chErr))
	cnl()

	wg.Wait()
	close(chErr)

	for err := range chErr {
		ee = append(ee, err)
	}
	return errors.Aggregate(ee...)
}

func (s *service) waitTermination(chErr <-chan error) error {
	for {
		select {
		case sig := <-s.termSig:
			log.Infof("signal %s received", sig.String())
			switch sig {
			case syscall.SIGHUP:
				s.sighupHandler()
			default:
				return nil
			}
		case err := <-chErr:
			log.Info("component error received")
			return err
		}
	}
}
