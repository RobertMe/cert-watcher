package controller

import (
	"context"
	"errors"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/RobertMe/cert-watcher/pkg/tracking"
	"github.com/RobertMe/cert-watcher/pkg/watcher"
	"github.com/rs/zerolog/log"
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

func (c *Controller) Start(parentCtx context.Context) {
	logger := log.Ctx(parentCtx).With().Str("component", "controller").Logger()
	ctx := logger.WithContext(parentCtx)
	go func() {
		<-ctx.Done()
		logger.Info().Msg("Received stop signal")

		c.Stop()
	}()

	logger.Info().Msg("Starting listeners")
	go c.listenWatchers(ctx)
	go c.listenSubscribers(ctx)

	watcherContext := createLoggerContext("watcher", parentCtx)
	subscriberContext := createLoggerContext("subscriber", parentCtx)

	c.tracker.Start(ctx)
	go c.watcher.Watch(c.watcherChan, watcherContext)
	go c.subscriber.Subscribe(c.subscriberChan, subscriberContext)
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
	logger := log.Ctx(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Stopping watcher listener")
			return
		case watcherMsg := <-c.watcherChan:
			c.tracker.CertificateChanged(&watcherMsg.Certificate)
		}
	}
}

func (c *Controller) listenSubscribers(ctx context.Context) {
	logger := log.Ctx(ctx)
	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Stopping subscribers listener")
			return
		case subscriberMsg := <-c.subscriberChan:
			if subscriberMsg.Action == subscriber.AddSubscriber {
				c.tracker.AddSubscription(subscriberMsg)
			}
		}
	}
}

func createLoggerContext(componentName string, parentCtx context.Context) context.Context {
	parentLogger := log.Ctx(parentCtx)
	logger := parentLogger.With().Str("component", componentName).Logger()
	return logger.WithContext(parentCtx)
}
