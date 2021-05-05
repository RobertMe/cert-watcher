package main

import (
	"context"
	"flag"
	"github.com/RobertMe/cert-watcher/pkg/config/static"
	"github.com/RobertMe/cert-watcher/pkg/controller"
	subscriberChain "github.com/RobertMe/cert-watcher/pkg/subscriber/chain"
	watcherChain "github.com/RobertMe/cert-watcher/pkg/watcher/chain"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	debug := flag.Bool("debug", false, "sets log level to debug")

	flag.Parse()

	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Info().Msg("Start cert-watcher")

	// TODO: flag
	config, err := static.ReadConfiguration("")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to read configuration file")
		return
	}

	log.Info().Interface("config", config).Msg("Loaded configuration")

	log.Debug().Msg("Creating watchers and subscribers")
	watchers := watcherChain.NewWatcherChain(*config.Watchers)
	subscribers := subscriberChain.NewSubscriberChain(*config.Subscribers)
	log.Debug().Msg("Created watchers and subscribers")

	ctx := createContext()
	ctx = log.Logger.WithContext(ctx)

	log.Debug().Msg("Creating controller")
	ctr := controller.NewController(watchers, subscribers)
	log.Debug().Msg("Created controller")

	log.Info().Msg("Starting controller")
	ctr.Start(ctx)
	log.Info().Msg("Started controller")

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
