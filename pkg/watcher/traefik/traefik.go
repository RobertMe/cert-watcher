package traefik

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/RobertMe/cert-watcher/pkg/cert"
	"github.com/RobertMe/cert-watcher/pkg/watcher"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/fsnotify/fsnotify.v1"
	"io/ioutil"
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

func (w *Watcher) Watch(certificateChannel chan<- watcher.Message, parentCtx context.Context) error {
	logger := log.Ctx(parentCtx).With().Str("watcher", "traefik").Logger()
	ctxLog := logger.WithContext(parentCtx)
	w.certificateChannel = certificateChannel

	var err error
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		ctx, cancel := context.WithCancel(ctxLog)
		defer cancel()

		for {
			select {
			case event := <-w.watcher.Events:
				if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
					w.updateWatch(filepath.Dir(event.Name), &logger)
				} else {
					w.updateWatch(w.AcmePath, &logger)
				}
				if event.Name == w.AcmePath &&
					(event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write) {
					w.readFile(&logger)
				}
				case err := <-w.watcher.Errors:
					logger.Error().Err(err).Msg("Error watching for acme.json changes")
				case <-ctx.Done():
					return
			}
		}
	}()

	w.updateWatch(w.AcmePath, &logger)

	go w.readFile(&logger)

	return nil
}

func (w *Watcher) readFile(parentLogger *zerolog.Logger) {
	logger := parentLogger.With().Str("acme_path", w.AcmePath).Logger()
	logger.Info().Msg("Reading acme.json file")
	content, err := ioutil.ReadFile(w.AcmePath)
	if err != nil {
		logger.Error().Err(err).Msg("Error reading acme.json file")
		return
	}

	var acme map[string]acmeProvider
	err = json.Unmarshal(content, &acme)
	if err != nil {
		logger.Error().Err(err).Msg("Error parsing acme.json file as JSON")
		return
	}

	for providerName, provider := range acme {
		providerLogger := logger.With().Str("acme_provider", providerName).Logger()
		for _, certificate := range provider.Certificates {
			certificateLogger := providerLogger.With().
				Str("main_domain", certificate.Domain.Main).
				Strs("sans", certificate.Domain.Sans).
				Logger()
			certificateLogger.Debug().Msg("Reading certificate")

			domains := []string{certificate.Domain.Main}
			if certificate.Domain.Sans != nil {
				domains = append(domains, certificate.Domain.Sans...)
			}

			decodedCert, err := base64.StdEncoding.DecodeString(certificate.Certificate)
			if err != nil {
				certificateLogger.Error().Err(err).Msg("Error decoding crt")
				continue
			}

			decodedKey, err := base64.StdEncoding.DecodeString(certificate.Key)
			if err != nil {
				certificateLogger.Error().Err(err).Msg("Error decoding key")
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

func (w *Watcher) updateWatch(path string, logger *zerolog.Logger) {
	if w.watching == w.AcmePath && path == w.AcmePath {
		return
	}

	_, err := os.Stat(path)
	for err != nil {
		path = filepath.Dir(path)
		_, err = os.Stat(path)
	}

	if w.watching == path {
		logger.Debug().Str("watch_path", path).Msg("Retained acme.json watch path")
		return
	}

	w.watcher.Add(path)
	w.watcher.Remove(w.watching)

	w.watching = path

	logger.Debug().Str("watch_path", path).Msg("Updated acme.json watch path")
}
