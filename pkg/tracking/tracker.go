package tracking

import (
	"context"
	"github.com/RobertMe/cert-watcher/pkg/cert"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"github.com/rs/zerolog/log"
	"strings"
)

type wildcard struct {
	domains []string
	certificate *cert.Certificate
}

type Tracker struct {
	items map[string]*item

	wildcards map[string]*wildcard
	unmatchedSubscribers  map[string]*item

	certificateChangedChan chan *cert.Certificate
	addSubscriptionChan chan subscriber.Message
}

func NewTracker() *Tracker {
	return &Tracker{
		items:     make(map[string]*item),

		wildcards: make(map[string]*wildcard),
		unmatchedSubscribers: make(map[string]*item),

		certificateChangedChan: make(chan *cert.Certificate, 100),
		addSubscriptionChan: make(chan subscriber.Message, 100),
	}
}

func (t *Tracker) Start(parentCtx context.Context) {
	logger := log.Logger.With().Str("component", "tracker").Logger()
	ctx := logger.WithContext(parentCtx)
	go func() {
		for {
			select {
			case certificate := <- t.certificateChangedChan:
				t.certificateChanged(certificate, ctx)
			case message := <- t.addSubscriptionChan:
				switch message.Action {
				case subscriber.AddSubscriber:
					t.addSubscription(message, ctx)
					break
				}
			}
		}
	}()
}

func (t *Tracker) CertificateChanged(certificate *cert.Certificate) {
	t.certificateChangedChan <- certificate
}

func (t *Tracker) AddSubscription(message subscriber.Message) {
	t.addSubscriptionChan <- message
}

func (t *Tracker) certificateChanged(certificate *cert.Certificate, ctx context.Context) {
	logger := log.Ctx(ctx)
	logger.Info().Strs("names", certificate.Names).Msg("Handling changed certificate")
	for _, name := range certificate.Names {
		if strings.HasPrefix(name, "*.") {
			if wildcrd, ok := t.wildcards[name]; ok {
				for _, domain := range wildcrd.domains {
					t.items[domain].updateCertificate(certificate)
				}
			} else {
				t.wildcards[name] = &wildcard{
					domains: []string{},
					certificate: certificate,
				}
			}
		} else if item, ok := t.items[name]; ok {
			item.updateCertificate(certificate)
		} else {
			item := newItem(name, logger)
			item.certificate = certificate
			t.items[name] = item
		}
	}
}

func (t *Tracker) addSubscription(message subscriber.Message, ctx context.Context) {
	logger := log.Ctx(ctx)
	logger.Info().Strs("domains", message.Domains).Msg("Adding subscription")
	for _, domain := range message.Domains {
		if item, ok := t.items[domain]; ok {
			item.addSubscriber(message)

			return
		}

		wildcardName := "*" + domain[strings.Index(domain, "."):]

		wildcrd, ok := t.wildcards[wildcardName]
		logger.Debug().Bool("wildcard_found", ok).Str("wildcard_name", wildcardName).Msg("Checked wildcard")
		if !ok {
			wildcrd = &wildcard{}
			t.wildcards[wildcardName] = wildcrd
		}

		wildcrd.domains = append(wildcrd.domains, domain)

		item := newItem(domain, logger)
		t.items[domain] = item
		item.certificate = wildcrd.certificate
		item.addSubscriber(message)
	}
}
