package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"io"

	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/google/go-cmp/cmp"
)

const (
	pkgName        = "Docker. "
	cmdServiceInfo = "SERVICE_INFO"
	serviceType    = "DOCKER"
)

type dockerWatcher struct {
	writer         io.Writer
	cli            *client.Client
	ctx            context.Context
	serviceInfoMsg common.ServiceInfoMessage
}

func New(w io.Writer) DockerService {
	obj := &dockerWatcher{writer: w}

	obj.serviceInfoMsg.ID = env.MessageDefaultID
	obj.serviceInfoMsg.MsgType = cmdServiceInfo

	return obj
}

func (obj *dockerWatcher) Name() string {
	return cmdServiceInfo
}

func (obj *dockerWatcher) run() {
	// cleanup context and doker client on exit
	defer func() {
		obj.ctx = nil
		obj.cli = nil
	}()

	msgs, errs := obj.cli.Events(obj.ctx, types.EventsOptions{})

	for {
		select {
		case err, ok := <-errs:

			if !ok {
				return
			}
			logger.Error().Println(pkgName, "Error channel: ", err)
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			switch msg.Type {

			case events.ContainerEventType:
				if msg.Action == "create" || msg.Action == "destroy" ||
					msg.Action == "start" || msg.Action == "stop" {
					data := obj.ContainerInfo()

					if !cmp.Equal(data, obj.serviceInfoMsg.Data) {
						obj.serviceInfoMsg.Data = data
						obj.serviceInfoMsg.Now()

						raw, err := json.Marshal(obj.serviceInfoMsg)
						if err == nil {
							logger.Message().Println(pkgName, "Sending: ", string(raw))
							_, err = obj.writer.Write(raw)
						}
						if err != nil {
							logger.Error().Println(pkgName, "event", msg.Type, msg.Action, err)
						}
					}
				}

			default:
				logger.Debug().Println(pkgName, "Unhandled message", msg.Type, msg.Action)
			}
		}
	}
}

func (obj *dockerWatcher) Run(ctx context.Context) error {
	if obj.ctx != nil {
		return fmt.Errorf("docker watcher already running")
	}

	var err error
	obj.ctx = ctx
	obj.cli, err = client.NewClientWithOpts(client.FromEnv)
	if err == nil {
		obj.cli.NegotiateAPIVersion(ctx)
		logger.Info().Println(pkgName, "negotiated API version", obj.cli.ClientVersion())
	} else {
		logger.Error().Println(pkgName, "Docker client init: ", err)
		return fmt.Errorf("could not initialise docker client")
	}

	go obj.run()

	return nil
}
