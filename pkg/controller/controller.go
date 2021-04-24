package controller

import (
	"context"
	"errors"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/RobertMe/cert-watcher/pkg/tracking"
	"github.com/RobertMe/cert-watcher/pkg/watcher"
	"time"
)

type Controller struct {
	watcher    watcher.Watcher
	subscriber subscriber.Subscriber
	tracker    *tracking.Tracker

	watcherChan chan watcher.Message
	subscriberChan chan subscriber.Message

	stopChannel chan bool
}

func NewController(wtcr watcher.Watcher, subscr subscriber.Subscriber) *Controller {
	return &Controller{
		watcher:     wtcr,
		subscriber:  subscr,
		tracker:	 tracking.NewTracker(),

		watcherChan: make(chan watcher.Message, 100),
		subscriberChan: make(chan subscriber.Message, 100),

		stopChannel: make(chan bool, 1),
	}
}

func (c *Controller) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()

		c.Stop()
	}()

	go c.listenWatchers(ctx)
	go c.listenSubscribers(ctx)

	c.tracker.Start()
	go c.watcher.Watch(c.watcherChan)
	go c.subscriber.Subscribe(c.subscriberChan, ctx)
}

func (c *Controller) Stop() {
	c.stopChannel <- true
}

func (c *Controller) Wait() {
	<-c.stopChannel
}

func (c *Controller) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	go func(ctx context.Context) {
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			panic("Timeout while stopping cert-watcher, killing instance ✝")
		}
	}(ctx)

	close(c.stopChannel)

	cancel()
}

func (c *Controller) listenWatchers(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case watcherMsg := <-c.watcherChan:
			c.tracker.CertificateChanged(&watcherMsg.Certificate)
		}
	}
}

func (c *Controller) listenSubscribers(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case subscriberMsg := <-c.subscriberChan:
			if subscriberMsg.Action == subscriber.AddSubscriber {
				c.tracker.AddSubscription(subscriberMsg)
			}
		}
	}
}
