package docker

import (
	"context"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

func (s *Subscriber) getClientOptions() ([]client.Opt, error) {
	options := []client.Opt{
		client.WithHost(s.Endpoint),
		client.WithTimeout(s.ClientTimeout),
	}

	return options, nil
}

func (s *Subscriber) createClient() (client.APIClient, error) {
	options, err := s.getClientOptions()
	if err != nil {
		return nil, err
	}

	httpHeaders := map[string]string{
		"User-Agent": "cert-watcher",
	}
	options = append(options, client.WithHTTPHeaders(httpHeaders))

	options = append(options, client.WithVersion("1.24"))

	return client.NewClientWithOpts(options...)
}

func (s *Subscriber) listContainers(client client.APIClient, ctx context.Context) error {
	logger := log.Ctx(ctx)
	containers, err := client.ContainerList(ctx, dockertypes.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, container := range containers {
		config, ok := parseContainer(container.Labels)
		containerLogger := logger.With().
			Strs("container", container.Names).
			Interface("container_labels", container.Labels).
			Bool("ok", ok).
			Logger()
		if !ok {
			containerLogger.Debug().Msg("Parsed container, no valid configuration found")
			continue
		}

		containerLogger.Debug().
			Interface("configuration", config).
			Msg("Parsed container, valid configuration found")

		s.addContainer(container.ID, config)
	}

	return nil
}

func (s *Subscriber) listenContainers(client client.APIClient, ctx context.Context) {
	f := filters.NewArgs()
	f.Add("type", events.ContainerEventType)

	eventsChan, errChan := client.Events(ctx, dockertypes.EventsOptions{Filters: f})

	for {
		select {
		case event := <-eventsChan:
			switch event.Action {
			case "start":
				s.handleStart(event, client, ctx)
			case "die":
				s.handleStop(event, ctx)
			}
		case <-errChan:
		}
	}
}

func (s *Subscriber) handleStart(event events.Message, client client.APIClient, ctx context.Context) {
	logger := log.Ctx(ctx)

	container, err := client.ContainerInspect(ctx, event.ID)
	if err != nil {
		logger.Error().Err(err).Str("container_id", event.ID).Msg("Failed to introspect new container")
		return
	}

	if blockUntil, ok := s.blockUpdate[container.ID]; ok && (blockUntil == 0 || blockUntil > event.TimeNano) {
		return
	}

	config, ok := parseContainer(container.Config.Labels)
	containerLogger := logger.With().
		Strs("container", []string{container.Name}).
		Interface("container_labels", container.Config.Labels).
		Bool("ok", ok).
		Logger()

	if !ok {
		containerLogger.Debug().Msg("Parsed container, no valid configuration found")
		return
	}

	containerLogger.Debug().
		Interface("configuration", config).
		Msg("Parsed container, valid configuration found")

	s.addContainer(container.ID, config)
}

func (s *Subscriber) handleStop(event events.Message, ctx context.Context) {
	logger := log.Ctx(ctx)

	containerId := event.ID

	if _, ok := s.registeredContainers[containerId]; !ok {
		logger.Debug().Str("container_id", containerId).Msg("Ignoring stop of unregistered container")
		return
	}

	if blockUntil, ok := s.blockUpdate[containerId]; ok && (blockUntil == 0 || blockUntil > event.TimeNano) {
		return
	}

	delete(s.registeredContainers, containerId)
	delete(s.blockUpdate, containerId)

	logger.Debug().Str("container_id", containerId).Msg("Removed stopped container")
}
