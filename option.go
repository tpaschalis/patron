package patron

import (
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
)

type optionFunc func(*service) error

func sighub(handler func()) optionFunc {
	return func(s *service) error {
		if handler == nil {
			return errors.New("handler is nil")
		}
		s.sighupHandler = handler
		log.Info("SIGHUP handler set")
		return nil
	}
}
