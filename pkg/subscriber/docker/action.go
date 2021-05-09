package docker

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/cenkalti/backoff/v4"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type onErrorHandling int

const (
	onErrorRetry onErrorHandling = iota
	onErrorStop
	onErrorContinue
	onErrorRestartStop
	onErrorRestartContinue
)

type action interface {
	onError() onErrorHandling
	execute(invocation subscriber.Invocation, containerId string, client client.APIClient, ctx context.Context) error
}

func parseActionLabels(labels map[string]string) ([]action, bool) {
	var actionsData = map[int]map[string]string{}
	actionKeys := []int{}
	actionMatcher := regexp.MustCompile("^cert-watcher\\.actions\\[(\\d)+\\](?:\\.(.+))?$")
	for k, v := range labels {
		match := actionMatcher.FindStringSubmatch(k)
		if match == nil {
			continue
		}

		index, _ := strconv.Atoi(match[1])

		if _, ok := actionsData[index]; !ok {
			actionsData[index] = map[string]string{}
			actionKeys = append(actionKeys, index)
		}

		if match[2] != "" {
			actionsData[index][match[2]] = v
		} else {
			actionsData[index]["action_type"] = v
		}
	}

	if len(actionKeys) == 0 {
		return []action{}, false
	}

	sort.Ints(actionKeys)

	actions := []action{}
	for k := range actionKeys {
		actionData := actionsData[k]
		actionType, ok := actionData["action_type"]
		if !ok {
			return []action{}, false
		}
		delete(actionData, "action_type")

		var a action
		switch actionType {
		case "copy":
			a = newCopyAction(actionData)
		case "exec":
			a = newExecAction(actionData)
		case "restart":
			a = newRestartAction(actionData)
		default:
			return []action{}, false
		}

		if a == nil {
			return []action{}, false
		}

		actions = append(actions, a)
	}

	return actions, true
}

func (s *Subscriber) invokeActions(msg subscriber.Invocation, ctx context.Context) {
	containerId := msg.Data.(string)

	currentActionIndex := 0
	container := s.registeredContainers[containerId]
	actions := container.Actions

	logger := log.Ctx(ctx).With().
		Str("container_id", containerId).
		Str("domain", msg.Domain).
		Logger()

	logger.Info().Msg("Invoking actions on container")

	operation := func() error {
		if _, ok := s.registeredContainers[containerId]; !ok {
			logger.Info().Msg("Container doesn't exist anymore, stopping actions")
			return nil
		}

		client, err := s.createClient()
		if err != nil {
			logger.Error().Err(err).Msg("Failed connecting to docker daemon")
			return err
		}

		s.blockUpdate[containerId] = 0
		defer func() {
			s.blockUpdate[containerId] = time.Now().UnixNano()
		}()

		for ; currentActionIndex < len(actions); currentActionIndex++ {
			action := actions[currentActionIndex]

			actionLogger := logger.With().Interface("action", action).Logger()
			actionCtx := actionLogger.WithContext(ctx)

			err := action.execute(msg, containerId, client, actionCtx)
			if err == nil {
				actionLogger.Debug().Interface("action", action).Msg("Successfully executed action")
				continue
			}

			switch action.onError() {
			case onErrorRetry:
				actionLogger.Error().Err(err).Msg("Failed invoking action, retying later")
				return err
			case onErrorStop:
				actionLogger.Error().Err(err).Msg("Failed invoking action, stopping all actions")
				return nil
			case onErrorContinue:
				actionLogger.Error().Err(err).Msg("Failed invoking action, continuing with next action")
				continue
			case onErrorRestartStop:
				restartErr := restartContainerOnError(ctx, client, containerId)
				if restartErr != nil {
					logger.Error().
						Err(err).
						AnErr("restart_error", restartErr).
						Msg("Tried restarting container on error, but failed as well. Retrying later.")
					return err
				}

				actionLogger.Error().Err(err).Msg("Failed invoking action, restarted container and stopping all actions")
				return nil
			case onErrorRestartContinue:
				restartErr := restartContainerOnError(ctx, client, containerId)
				if restartErr != nil {
					logger.Error().
						Err(err).
						AnErr("restart_error", restartErr).
						Msg("Tried restarting container on error, but failed as well. Retrying later.")
					return err
				}

				actionLogger.Error().Err(err).Msg("Failed invoking action, restarted container and continuing with next action")
				continue
			}
		}

		return nil
	}

	notify := func(err error, time time.Duration) {
		logger.Error().Err(err).Dur("retry_at", time).Msg("Executing actions failed, retrying later")
	}

	backOff := backoff.NewExponentialBackOff()
	backOff.InitialInterval = 1 * time.Second
	err := backoff.RetryNotify(
		operation,
		backOff,
		notify,
	)
	if err != nil {
		logger.Error().Err(err).Msg("Executing actions failed permanently, not retrying")
	}
}

func parseActionOnError(actionData map[string]string) onErrorHandling {
	if onErrorValue, ok := actionData["on-error"]; ok {
		switch strings.ToLower(onErrorValue) {
		case "retry":
			return onErrorRetry
		case "stop":
			return onErrorStop
		case "continue":
			return onErrorContinue
		case "restart-stop":
			return onErrorRestartStop
		case "restart-continue":
			return onErrorRestartContinue
		}
	}

	return onErrorRetry
}

func restartContainerOnError(ctx context.Context, client client.APIClient, containerId string) error {
	timeout := 10 * time.Second
	return client.ContainerRestart(ctx, containerId, &timeout)
}
