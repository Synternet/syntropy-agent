package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

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

type DockerWatcher struct {
	slock.AtomicServiceLock
	writer io.Writer
	cli    *client.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func New(w io.Writer) *DockerWatcher {
	var err error
	dw := DockerWatcher{writer: w}
	dw.cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		// TODO: If docker is not running at startup - add period check if docker service has started
		logger.Error().Println(pkgName, "Docker client: ", err)
		dw.cli = nil // Just to make sure
	}

	return &dw
}

func (obj *DockerWatcher) Name() string {
	return cmdNetwork + " / " + cmdContainer
}

func (obj *DockerWatcher) run() {
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
					resp.ID = "-"
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
					resp.ID = "-"
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

func (obj *DockerWatcher) Start() (err error) {
	if !obj.TryLock() {
		return fmt.Errorf("docker watcher already running")
	}

	obj.ctx, obj.cancel = context.WithCancel(context.Background())

	go obj.run()

	return nil
}

func (obj *DockerWatcher) Stop() error {
	if !obj.TryUnlock() {
		return fmt.Errorf("docker watcher is not running")
	}

	obj.cancel()

	obj.cli = nil

	return nil
}
