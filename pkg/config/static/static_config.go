package static

import (
	"github.com/RobertMe/cert-watcher/pkg/subscriber/docker"
	"github.com/RobertMe/cert-watcher/pkg/watcher/traefik"
)

type Watchers struct {
	Traefik *traefik.Watcher `description:"Enable Traefik watcher" json:"traefik" yaml:"traefik"`
}

type Subscribers struct {
	Docker *docker.Subscriber `description:"Enable Docker subscriber" json:"docker" yaml:"docker"`
}

type Configuration struct {
	Watchers    *Watchers    `description:"Watchers configuration" json:"watchers" yaml:"watchers"`
	Subscribers *Subscribers `description:"Subscribers configuration" json:"subscribers" yaml:"subscribers"`
}

func NewConfiguration() *Configuration {
	return &Configuration{
		Watchers:    &Watchers{},
		Subscribers: &Subscribers{},
	}
}
