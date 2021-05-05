package watcher

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/cert"
)

type Message struct {
	MonitorName string
	Certificate cert.Certificate
}

type Watcher interface {
	Init() error
	Watch(certificateChannel chan<- Message, parentCtx context.Context) error
}
