package agent

import (
	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

func (a *Agent) addService(s common.Service) error {
	a.services = append(a.services, s)
	return nil
}

func (a *Agent) startServices() {
	for _, s := range a.services {
		logger.Info().Printf("%s Starting %s service.\n", pkgName, s.Name())
		err := s.Run(a.ctx)
		if err != nil {
			logger.Error().Printf("%s Service %s: %s\n", pkgName, s.Name(), err.Error())
		}
	}
}

func (a *Agent) stopServices() {
	logger.Info().Printf("%s Stopping services.\n", pkgName)
	a.cancel()
}
