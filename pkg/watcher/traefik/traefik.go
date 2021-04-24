package traefik

import (
	"encoding/base64"
	"encoding/json"
	"github.com/RobertMe/cert-watcher/pkg/cert"
	"github.com/RobertMe/cert-watcher/pkg/watcher"
	"gopkg.in/fsnotify/fsnotify.v1"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type Watcher struct {
	AcmePath string `description:"Path to the acme.json file" json:"acme_path" yaml:"acme_path"`

	certificateChannel chan<- watcher.Message
	watcher *fsnotify.Watcher
	watching string
}

func (w *Watcher) Init() error {
	return nil
}

func (w *Watcher) Watch(certificateChannel chan<- watcher.Message) error {
	w.certificateChannel = certificateChannel

	var err error
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-w.watcher.Events:
				if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
					w.updateWatch(filepath.Dir(event.Name))
				} else {
					w.updateWatch(w.AcmePath)
				}
				if event.Name == w.AcmePath &&
					(event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write) {
					w.readFile()
				}
				case <-w.watcher.Errors:
					// TODO: log
			}
		}
	}()

	w.updateWatch(w.AcmePath)

	go w.readFile()

	return nil
}

func (w *Watcher) readFile() {
	log.Println("Reading")
	content, err := ioutil.ReadFile(w.AcmePath)
	if err != nil {
		// TODO: log
		return
	}

	var acme map[string]acmeProvider
	err = json.Unmarshal(content, &acme)
	if err != nil {
		// TODO: log
		return
	}

	for _, provider := range acme {
		for _, certificate := range provider.Certificates {
			domains := []string{certificate.Domain.Main}
			if certificate.Domain.Sans != nil {
				domains = append(domains, certificate.Domain.Sans...)
			}

			decodedCert, err := base64.StdEncoding.DecodeString(certificate.Certificate)
			if err != nil {
				// TODO: log
				continue
			}

			decodedKey, err := base64.StdEncoding.DecodeString(certificate.Key)
			if err != nil {
				// TODO: log
				continue
			}

			certFile := cert.Certificate{
				Names: domains,
				Cert:  decodedCert,
				Key:   decodedKey,
			}

			w.certificateChannel <- watcher.Message{
				MonitorName: "traefik",
				Certificate: certFile,
			}
		}
	}
}

func (w *Watcher) updateWatch(path string) {
	if w.watching == w.AcmePath && path == w.AcmePath {
		return
	}

	_, err := os.Stat(path)
	for err != nil {
		path = filepath.Dir(path)
		_, err = os.Stat(path)
	}

	if w.watching == path {
		return
	}

	w.watcher.Add(path)
	w.watcher.Remove(w.watching)

	w.watching = path
}
