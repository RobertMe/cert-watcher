package tracking

import (
	"crypto/sha1"
	"github.com/RobertMe/cert-watcher/pkg/cert"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"log"
)

type item struct {
	domain      string
	certificate *cert.Certificate
	subscribers []subscriber.Message
	sum         [sha1.Size]byte
}

func newItem(domain string) *item {
	return &item{
		domain:      domain,
		certificate: nil,
		subscribers: []subscriber.Message{},
	}
}

func (i *item) updateCertificate(certificate *cert.Certificate) {
	sum := sha1.Sum(certificate.Cert)
	if i.sum == sum {
		return
	}

	i.certificate = certificate
	i.sum = sum

	for _, subscr := range i.subscribers {
		log.Println("Invoking subscriber")
		subscr.Channel <- subscriber.Invocation{
			Domain:      i.domain,
			Certificate: *certificate,
			Data:        subscr.UpdateData,
		}
	}
}

func (i *item) addSubscriber(message subscriber.Message) {
	i.subscribers = append(i.subscribers, message)

	if i.certificate != nil {
		log.Println("Invoking subscriber")
		message.Channel <- subscriber.Invocation{
			Domain:      i.domain,
			Certificate: *i.certificate,
			Data:        message.UpdateData,
		}
	}
}
