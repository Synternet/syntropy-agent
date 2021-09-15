package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
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
	slock.AtomicServiceLock
	writer io.Writer
	cli    *client.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func New(w io.Writer) DockerService {
	var err error
	dw := dockerWatcher{writer: w}
	dw.ctx, dw.cancel = context.WithCancel(context.Background())
	dw.cli, err = client.NewClientWithOpts(client.FromEnv)
	if err == nil {
		dw.cli.NegotiateAPIVersion(dw.ctx)
		logger.Info().Println(pkgName, "negotiated API version", dw.cli.ClientVersion())
	} else {
		logger.Error().Println(pkgName, "Docker client: ", err)
		logger.Warning().Println(pkgName, "fallback to null Docker client")
		return &DockerNull{}
	}

	return &dw
}

func (obj *dockerWatcher) Name() string {
	return cmdNetwork + " / " + cmdContainer
}

func (obj *dockerWatcher) run() {
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

func (obj *dockerWatcher) Start() (err error) {
	if !obj.TryLock() {
		return fmt.Errorf("docker watcher already running")
	}

	go obj.run()

	return nil
}

func (obj *dockerWatcher) Stop() error {
	if !obj.TryUnlock() {
		return fmt.Errorf("docker watcher is not running")
	}

	obj.cancel()

	return nil
}
