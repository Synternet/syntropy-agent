package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

const (
	pkgName      = "Docker. "
	cmdNetwork   = "NETWORK_INFO"
	cmdContainer = "CONTAINER_INFO"
)

type dockerWatcher struct {
	writer io.Writer
	cli    *client.Client
	ctx    context.Context
}

func New(w io.Writer) DockerService {
	return &dockerWatcher{writer: w}
}

func (obj *dockerWatcher) Name() string {
	return cmdNetwork + " / " + cmdContainer
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
			case events.NetworkEventType:

				if msg.Action == "create" || msg.Action == "destroy" {
					resp := networkInfoMessage{
						Data: obj.NetworkInfo(),
					}
					resp.ID = env.MessageDefaultID
					resp.MsgType = cmdNetwork
					resp.Now()
					raw, err := json.Marshal(resp)
					if err == nil {
						logger.Debug().Println(pkgName, "Sending: ", string(raw))
						_, err = obj.writer.Write(raw)
					}
					if err != nil {
						logger.Error().Println(pkgName, "event", msg.Type, msg.Action, err)
					}
				}
			case events.ContainerEventType:
				if msg.Action == "create" || msg.Action == "destroy" ||
					msg.Action == "start" || msg.Action == "stop" {
					resp := containerInfoMessage{
						Data: obj.ContainerInfo(),
					}
					resp.ID = env.MessageDefaultID
					resp.MsgType = cmdContainer
					resp.Now()
					raw, err := json.Marshal(resp)
					if err == nil {
						logger.Debug().Println(pkgName, "Sending: ", string(raw))
						_, err = obj.writer.Write(raw)
					}
					if err != nil {
						logger.Error().Println(pkgName, "event", msg.Type, msg.Action, err)
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
