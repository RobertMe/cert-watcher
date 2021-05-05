package tracking

import (
	"crypto/sha1"
	"github.com/RobertMe/cert-watcher/pkg/cert"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/rs/zerolog"
)

type item struct {
	domain      string
	certificate *cert.Certificate
	subscribers []subscriber.Message
	sum         [sha1.Size]byte
	logger      zerolog.Logger
}

func newItem(domain string, parentLogger *zerolog.Logger) *item {
	return &item{
		domain:      domain,
		certificate: nil,
		subscribers: []subscriber.Message{},
		logger:      parentLogger.With().Str("certificate", domain).Logger(),
	}
}

func (i *item) updateCertificate(certificate *cert.Certificate) {
	sum := sha1.Sum(certificate.Cert)
	if i.sum == sum {
		i.logger.Info().Msg("Skipping certificate update as it didn't change")
		return
	}

	i.certificate = certificate
	i.sum = sum

	for _, subscr := range i.subscribers {
		i.invokeSubscriber(subscr)
	}
}

func (i *item) addSubscriber(message subscriber.Message) {
	i.subscribers = append(i.subscribers, message)

	if i.certificate != nil {
		i.invokeSubscriber(message)
	}
}

func (i *item) invokeSubscriber(subscr subscriber.Message) {
	i.logger.Info().Str("subscriber", subscr.SubscriberName).Msg("Invoking subscriber")
	subscr.Channel <- subscriber.Invocation{
		Domain:      i.domain,
		Certificate: *i.certificate,
		Data:        subscr.UpdateData,
	}
}
