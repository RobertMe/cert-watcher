package docker

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/cenkalti/backoff/v4"
	"github.com/docker/docker/client"
	"log"
	"regexp"
	"strings"
	"time"
)

type configuration struct {
	domains []string
	actions []action
}

type Subscriber struct {
	Endpoint string
	ClientTimeout time.Duration

	registeredContainers map[string]configuration
	blockUpdate []string
	unblockUpdate []string

	subscriptionChannel chan<- subscriber.Message
	channel chan subscriber.Invocation
}

func (s *Subscriber) Init() error {
	if s.Endpoint == "" {
		s.Endpoint = client.DefaultDockerHost
	}

	s.registeredContainers = map[string]configuration{}

	s.channel = make(chan subscriber.Invocation, 10)

	return nil
}

func (s *Subscriber) Subscribe(subscriptionChannel chan<- subscriber.Message, parentCtx context.Context) error {
	s.subscriptionChannel = subscriptionChannel

	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				return
			case msg := <- s.channel:
				s.invokeActions(msg, ctx)
			}
		}
	}(parentCtx)

	go func() {
		operation := func() error {
			ctx, cancel := context.WithCancel(parentCtx)
			defer cancel()

			client, err := s.createClient()
			if err != nil {
				// TODO: log
				return err
			}

			s.listContainers(client, ctx)

			s.listenContainers(client, ctx)

			return nil
		}

		notify := func(err error, time time.Duration) {
			log.Println(err)
			// TODO: log
		}
		err := backoff.RetryNotify(
			operation,
			backoff.NewExponentialBackOff(),
			notify,
		)
		if err != nil {
			log.Println(err)
			// TODO: log
		}
	}()

	return nil
}

func parseContainer(labels map[string]string) (configuration, bool) {
	config := configuration{
		actions: []action{},
	}
	domainsLabel, ok := labels["cert-watcher.domains"]
	if !ok {
		return config, false
	}

	splitter := regexp.MustCompile("\\s*,(\\s*,*)*")
	config.domains = splitter.Split(strings.TrimSpace(domainsLabel), -1)

	if len(config.domains) == 0 {
		return config, false
	}

	if config.actions, ok = parseActionLabels(labels); !ok {
		return config, false
	}

	return config, true
}

func (s *Subscriber) addContainer(containerId string, config configuration) {
	msg := subscriber.Message{
		SubscriberName: "docker",
		Action:         subscriber.AddSubscriber,
		Domains:        config.domains,
		UpdateData:     containerId,
		Channel:        s.channel,
	}

	s.subscriptionChannel <- msg

	s.registeredContainers[containerId] = config
}
