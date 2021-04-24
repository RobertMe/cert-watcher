package main

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/config/static"
	"github.com/RobertMe/cert-watcher/pkg/controller"
	chain2 "github.com/RobertMe/cert-watcher/pkg/subscriber/chain"
	"github.com/RobertMe/cert-watcher/pkg/watcher/chain"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// TODO: flag
	config, err := static.ReadConfiguration("")
	if err != nil {
		// TODO: log/show
		return
	}

	watchers := chain.NewWatcherChain(*config.Watchers)
	subscribers := chain2.NewSubscriberChain(*config.Subscribers)

	ctx := createContext()

	ctr := controller.NewController(watchers, subscribers)

	ctr.Start(ctx)

	ctr.Wait()
}

func createContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		cancel()
	}()

	return ctx
}
