package chain

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/config/static"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
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
		// TODO: log
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
	for _, subscr := range s.Subscribers {
		go startSubscriber(subscr, subscriptionChannel, parentCtx)
	}

	return nil
}

func startSubscriber(subscriber subscriber.Subscriber, subscriptionChannel chan<- subscriber.Message, parentCtx context.Context) {
	if err := subscriber.Subscribe(subscriptionChannel, parentCtx); err != nil {
		// TODO: log
	}
}
