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

type Log struct {
	Level string `description:"Log level" json:"level" yaml:"level"`
}

type Configuration struct {
	Watchers    *Watchers    `description:"Watchers configuration" json:"watchers" yaml:"watchers"`
	Subscribers *Subscribers `description:"Subscribers configuration" json:"subscribers" yaml:"subscribers"`
	Log         *Log         `description:"Logging configuration" json:"log" yaml:"log"`
}

func NewConfiguration() *Configuration {
	return &Configuration{
		Watchers:    &Watchers{},
		Subscribers: &Subscribers{},
		Log: &Log{
			Level: "ERROR",
		},
	}
}
