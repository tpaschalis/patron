package patron

import (
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/info"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/sync/http"
)

type optionFunc func(*service) error

func routes(rr []http.Route) optionFunc {
	return func(s *service) error {
		if len(rr) == 0 {
			return errors.New("routes are required")
		}
		s.routes = rr
		log.Info("routes options are set")
		return nil
	}
}

func middlewares(mm ...http.MiddlewareFunc) optionFunc {
	return func(s *service) error {
		if len(mm) == 0 {
			return errors.New("middlewares are required")
		}
		s.middlewares = mm
		log.Info("middleware options are set")
		return nil
	}
}

func healthCheck(hcf http.HealthCheckFunc) optionFunc {
	return func(s *service) error {
		if hcf == nil {
			return errors.New("health check func is required")
		}
		s.hcf = hcf
		log.Info("health check func is set")
		return nil
	}
}

func components(cc ...Component) optionFunc {
	return func(s *service) error {
		if len(cc) == 0 || cc[0] == nil {
			return errors.New("components are required")
		}
		s.cps = append(s.cps, cc...)
		log.Info("component options are set")
		return nil
	}
}

func docs(file string) optionFunc {
	return func(s *service) error {
		err := info.ImportDoc(file)
		if err != nil {
			return err
		}
		log.Info("documentation is set")
		return nil
	}
}

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
