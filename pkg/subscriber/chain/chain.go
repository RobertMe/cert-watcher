package chain

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/config/static"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/rs/zerolog/log"
	"reflect"
)

type SubscriberChain struct {
	Subscribers []subscriber.Subscriber
}

func NewSubscriberChain(conf static.Subscribers) *SubscriberChain {
	s := SubscriberChain{}

	if conf.Docker != nil {
		s.quietAddSubscriber(conf.Docker)
	}

	return &s
}

func (s *SubscriberChain) quietAddSubscriber(subscriber subscriber.Subscriber) {
	if err := s.AddSubscriber(subscriber); err != nil {
		log.Error().Err(err).Str("subscriber", reflect.TypeOf(subscriber).String()).Msg("Failed initializing subscriber")
	}
}

func (s *SubscriberChain) AddSubscriber(subscriber subscriber.Subscriber) error {
	if err := subscriber.Init(); err != nil {
		return err
	}

	s.Subscribers = append(s.Subscribers, subscriber)

	return nil
}

func (s *SubscriberChain) Init() error {
	return nil
}

func (s *SubscriberChain) Subscribe(subscriptionChannel chan<- subscriber.Message, parentCtx context.Context) error {
	logger := log.Ctx(parentCtx).With().Str("subscriber", "chain").Logger()
	for _, subscr := range s.Subscribers {
		go func(subscr subscriber.Subscriber) {
			if err := subscr.Subscribe(subscriptionChannel, parentCtx); err != nil {
				logger.Error().Err(err).Str("failed_subscriber", reflect.TypeOf(subscr).String()).Msg("Failed starting subscriber")
			}
		}(subscr)
	}

	return nil
}
