package docker

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/cenkalti/backoff/v4"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"regexp"
	"strings"
	"time"
)

type configuration struct {
	Domains []string
	Actions []action
}

type Subscriber struct {
	Endpoint string
	ClientTimeout time.Duration

	registeredContainers map[string]configuration
	blockUpdate map[string]int64

	subscriptionChannel chan<- subscriber.Message
	channel chan subscriber.Invocation
}

func (s *Subscriber) Init() error {
	if s.Endpoint == "" {
		s.Endpoint = client.DefaultDockerHost
	}

	s.registeredContainers = map[string]configuration{}
	s.blockUpdate = map[string]int64{}

	s.channel = make(chan subscriber.Invocation, 10)

	return nil
}

func (s *Subscriber) Subscribe(subscriptionChannel chan<- subscriber.Message, parentCtx context.Context) error {
	logger := log.Ctx(parentCtx).With().Str("subscriber", "docker").Logger()
	ctxLog := logger.WithContext(parentCtx)
	s.subscriptionChannel = subscriptionChannel

	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				logger.Info().Msg("Stopping subscriber")
				return
			case msg := <- s.channel:
				s.invokeActions(msg, ctx)
			}
		}
	}(ctxLog)

	go func() {
		operation := func() error {
			ctx, cancel := context.WithCancel(ctxLog)
			defer cancel()

			client, err := s.createClient()
			if err != nil {
				logger.Error().Err(err).Msg("Failed connecting to docker daemon")
				return err
			}

			if e := log.Debug(); e.Enabled() {
				if serverVersion, err := client.ServerVersion(ctx); err == nil {
					logger.Debug().
						Str("docker_version", serverVersion.Version).
						Str("docker_api_version", serverVersion.APIVersion).
						Msg("Connected to docker daemon")
				}
			}

			err = s.listContainers(client, ctx)
			if err != nil {
				logger.Error().Err(err).Msg("Failed listing containers")
				return err
			}

			s.listenContainers(client, ctx)

			return nil
		}

		notify := func(err error, time time.Duration) {
			logger.Error().Err(err).Dur("retry_at", time).Msg("Operation failed, retying later")
		}
		err := backoff.RetryNotify(
			operation,
			backoff.NewExponentialBackOff(),
			notify,
		)
		if err != nil {
			log.Error().Err(err).Msg("Operation failed permanently, not retrying")
		}
	}()

	return nil
}

func parseContainer(labels map[string]string) (configuration, bool) {
	config := configuration{
		Actions: []action{},
	}
	domainsLabel, ok := labels["cert-watcher.domains"]
	if !ok {
		return config, false
	}

	splitter := regexp.MustCompile("\\s*,(\\s*,*)*")
	config.Domains = splitter.Split(strings.TrimSpace(domainsLabel), -1)

	if len(config.Domains) == 0 {
		return config, false
	}

	if config.Actions, ok = parseActionLabels(labels); !ok {
		return config, false
	}

	return config, true
}

func (s *Subscriber) addContainer(containerId string, config configuration) {
	msg := subscriber.Message{
		SubscriberName: "docker",
		Action:         subscriber.AddSubscriber,
		Domains:        config.Domains,
		UpdateData:     containerId,
		Channel:        s.channel,
	}

	s.subscriptionChannel <- msg

	s.registeredContainers[containerId] = config
}
