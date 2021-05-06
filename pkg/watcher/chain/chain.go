package chain

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/config/static"
	"github.com/RobertMe/cert-watcher/pkg/watcher"
	"github.com/rs/zerolog/log"
	"reflect"
)

type WatcherChain struct {
	Watchers []watcher.Watcher
}

func NewWatcherChain(conf static.Watchers) *WatcherChain {
	w := WatcherChain{}

	if conf.Traefik != nil {
		w.quietAddWatcher(conf.Traefik)
	}

	return &w
}

func (w *WatcherChain) quietAddWatcher(watcher watcher.Watcher) {
	if err := w.AddWatcher(watcher); err != nil {
		log.Error().Err(err).Str("watcher", reflect.TypeOf(watcher).String()).Msg("Failed initializing watcher")
	}
}

func (w *WatcherChain) AddWatcher(watcher watcher.Watcher) error {
	if err := watcher.Init(); err != nil {
		return err
	}

	w.Watchers = append(w.Watchers, watcher)

	return nil
}

func (w *WatcherChain) Init() error {
	return nil
}

func (w *WatcherChain) Watch(certificateChannel chan<- watcher.Message, parentCtx context.Context) error {
	logger := log.Ctx(parentCtx).With().Str("watcher", "chain").Logger()
	for _, watch := range w.Watchers {
		go func(watch watcher.Watcher) {
			if err := watch.Watch(certificateChannel, parentCtx); err != nil {
				logger.Error().Err(err).Str("failed_watcher", reflect.TypeOf(watch).String()).Msg("Failed starting watcher")
			}
		}(watch)
	}
	return nil
}
