package agent

import (
	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

func (a *Agent) addService(s controller.Service) error {
	a.services = append(a.services, s)
	return nil
}

func (a *Agent) startServices() error {
	for _, s := range a.services {
		logger.Info().Printf("%s Starting %s service.\n", pkgName, s.Name())
		err := s.Start()
		if err != nil {
			logger.Error().Printf("%s Service %s: %s\n", pkgName, s.Name(), err.Error())
		}
	}
	return nil
}

func (a *Agent) stopServices() error {
	for _, s := range a.services {
		logger.Info().Printf("%s Stopping %s service.\n", pkgName, s.Name())
		err := s.Stop()
		if err != nil {
			logger.Error().Printf("%s Service %s: %s\n", pkgName, s.Name(), err.Error())
		}
	}

	return nil
}
