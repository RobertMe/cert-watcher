package chain

import (
	"github.com/RobertMe/cert-watcher/pkg/config/static"
	"github.com/RobertMe/cert-watcher/pkg/watcher"
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
		// TODO: log
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

func (w *WatcherChain) Watch(certificateChannel chan<- watcher.Message) error {
	for _, watch := range w.Watchers {
		go startWatcher(watch, certificateChannel)
	}
	return nil
}

func startWatcher(watcher watcher.Watcher, certificateChannel chan<- watcher.Message) {
	if err := watcher.Watch(certificateChannel); err != nil {
		// TODO: log
	}
}
