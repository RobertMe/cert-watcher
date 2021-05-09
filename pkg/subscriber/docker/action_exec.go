package docker

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"regexp"
	"sort"
	"strconv"
)

type actionExec struct {
	Command string
	Arguments []string
	User string
	WorkDir string
	OnError onErrorHandling
}

func newExecAction(data map[string]string) *actionExec {
	a := actionExec{
		OnError: parseActionOnError(data),
	}
	var ok bool
	if a.Command, ok = data["command"]; !ok {
		return nil
	}

	a.User = data["user"]
	a.WorkDir = data["workDir"]

	re := regexp.MustCompile("^args\\[(\\d+)\\]$")
	args := map[int]string{}
	var argIndexes []int
	for k, v := range data {
		match := re.FindStringSubmatch(k)
		if match == nil {
			continue
		}

		index, _ := strconv.Atoi(match[1])
		argIndexes = append(argIndexes, index)
		args[index] = v
	}

	sort.Ints(argIndexes)

	a.Arguments = make([]string, len(argIndexes))
	for i, k := range argIndexes {
		a.Arguments[i] = args[k]
	}

	return &a
}

func (a *actionExec) onError() onErrorHandling {
	return a.OnError
}

func (a *actionExec) execute(invocation subscriber.Invocation, containerId string, client client.APIClient, ctx context.Context) error {
	config := dockertypes.ExecConfig{
		Cmd: append([]string{a.Command}, a.Arguments...),
		User: a.User,
		WorkingDir: a.WorkDir,
		Detach: true,
	}

	response, err := client.ContainerExecCreate(ctx, containerId, config)
	if err != nil {
		return err
	}

	execStartCheck := dockertypes.ExecStartCheck{
		Detach: true,
		Tty:    false,
	}
	return client.ContainerExecStart(ctx, response.ID, execStartCheck)
}
