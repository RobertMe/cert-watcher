package docker

import (
	"context"
	"time"

	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type actionRestart struct {
	Timeout int
}

func newRestartAction(data map[string]string) *actionRestart {
	a := actionRestart{
		Timeout: 5,
	}

	if timeout, ok := data["timeout"]; ok {
		if duration, err := time.ParseDuration(timeout); err == nil {
			a.Timeout = int(duration.Seconds())
		}
	}

	return &a
}

func (a *actionRestart) execute(_ subscriber.Invocation, containerId string, client client.APIClient, ctx context.Context) error {
	return client.ContainerRestart(ctx, containerId, container.StopOptions{Timeout: &a.Timeout})
}
