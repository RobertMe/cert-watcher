package tracking

import (
	"github.com/RobertMe/cert-watcher/pkg/cert"
	"github.com/RobertMe/cert-watcher/pkg/subscriber"
	"log"
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

func (t *Tracker) Start() {
	go func() {
		for {
			select {
			case certificate := <- t.certificateChangedChan:
				t.certificateChanged(certificate)
			case message := <- t.addSubscriptionChan:
				t.addSubscription(message)
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

func (t *Tracker) certificateChanged(certificate *cert.Certificate) {
	for _, name := range certificate.Names {
		log.Println("Certificate changed for: ", name)
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
			item := newItem(name)
			item.certificate = certificate
			t.items[name] = item
		}
	}
}

func (t *Tracker) addSubscription(message subscriber.Message) {
	for _, domain := range message.Domains {
		log.Println("Subscription added for:", domain)
		if item, ok := t.items[domain]; ok {
			item.addSubscriber(message)

			return
		}

		wildcardName := "*" + domain[strings.Index(domain, "."):]

		wildcrd, ok := t.wildcards[wildcardName]
		log.Printf("Checked wildcard %s, result is %t", wildcardName, ok)
		if !ok {
			wildcrd = &wildcard{}
			t.wildcards[wildcardName] = wildcrd
		}

		wildcrd.domains = append(wildcrd.domains, domain)

		item := newItem(domain)
		t.items[domain] = item
		item.certificate = wildcrd.certificate
		item.addSubscriber(message)
	}
}
