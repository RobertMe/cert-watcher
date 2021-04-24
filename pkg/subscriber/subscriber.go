package subscriber

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/cert"
)

const (
	AddSubscriber    = "add_subscriber"
	RemoveSubscriber = "remove_subscriber"
)

type Invocation struct {
	Domain 		string
	Certificate cert.Certificate
	Data        interface{}
}

type Message struct {
	SubscriberName string
	Action         string
	Domains        []string
	UpdateData     interface{}
	Channel        chan<- Invocation
}

type Subscriber interface {
	Init() error
	Subscribe(subscriptionChannel chan<- Message, parentCtx context.Context) error
}
