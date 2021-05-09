package docker

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/docker/docker/client"
	"time"
)

type actionRestart struct {
	OnError onErrorHandling
	Timeout time.Duration
}

func newRestartAction(data map[string]string) *actionRestart {
	a := actionRestart{
		OnError: parseActionOnError(data),
		Timeout: 5 * time.Second,
	}

	if timeout, ok := data["timeout"]; ok {
		if duration, err := time.ParseDuration(timeout); err == nil {
			a.Timeout = duration
		}
	}

	return &a
}

func (a *actionRestart) onError() onErrorHandling {
	return a.OnError
}

func (a *actionRestart) execute(_ subscriber.Invocation, containerId string, client client.APIClient, ctx context.Context) error {
	return client.ContainerRestart(ctx, containerId, &a.Timeout)
}
