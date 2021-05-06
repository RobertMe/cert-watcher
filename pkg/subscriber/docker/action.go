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
	"time"
)

type action interface {
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
		client, err := s.createClient()
		if err != nil {
			logger.Error().Err(err).Msg("Failed connecting to docker daemon")
			return err
		}

		s.blockUpdate = append(s.blockUpdate, containerId)
		defer func() {
			s.unblockUpdate = append(s.unblockUpdate, containerId)
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

			actionLogger.Error().Err(err).Msg("Failed invoking action")
		}

		return nil
	}

	notify := func(err error, time time.Duration) {
		logger.Error().Err(err).Dur("retry_at", time).Msg("Executing actions failed, retrying later")
	}

	err := backoff.RetryNotify(
		operation,
		backoff.NewExponentialBackOff(),
		notify,
	)
	if err != nil {
		logger.Error().Err(err).Msg("Executing action failed permanently, not retrying")
	}
}
