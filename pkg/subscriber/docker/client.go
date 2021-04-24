package docker

import (
	"context"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"log"
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
	containers, err := client.ContainerList(ctx, dockertypes.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, container := range containers {
		config, ok := parseContainer(container.Labels)
		log.Println(config, ok)
		if !ok {
			continue
		}

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

			}
			case <-errChan:
		}
	}
}

func (s *Subscriber) handleStart(event events.Message, client client.APIClient, ctx context.Context) {
	defer func(containerId string) {
		for i, id := range s.unblockUpdate {
			if id == containerId {
				s.unblockUpdate = append(s.unblockUpdate[:i], s.unblockUpdate[i+1:]...)

				for i, id := range s.blockUpdate {
					if id == containerId {
						s.blockUpdate = append(s.blockUpdate[:i], s.blockUpdate[i+1:]...)
						break
					}
				}

				break
			}
		}
	}(event.ID)

	container, err := client.ContainerInspect(ctx, event.ID)
	if err != nil {
		// TODO: log
		return
	}

	for _, id := range s.blockUpdate {
		if id == container.ID {
			return
		}
	}

	config, ok := parseContainer(container.Config.Labels)
	log.Println(config, ok)
	if !ok {
		return
	}

	s.addContainer(container.ID, config)
}
