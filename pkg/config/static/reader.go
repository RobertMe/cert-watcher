package static

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

func ReadConfiguration(configFile string) (*Configuration, error) {
	config := NewConfiguration()

	path, err := findConfig(configFile)
	if err != nil {
		return nil, err
	}

	if path == "" {
		return config, nil
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(content, config); err != nil {
		return nil, err
	}

	return config, nil
}

func findConfig(configFile string) (string, error) {
	for _, path := range getPaths(configFile) {
		path := os.ExpandEnv(path)

		info, err := os.Stat(path)

		if os.IsNotExist(err) || info.IsDir() {
			continue
		}

		if err != nil {
			return "", err
		}

		return filepath.Abs(path)
	}

	return "", nil
}

func getPaths(configFile string) []string {
	var paths []string

	if strings.TrimSpace(configFile) != "" {
		paths = append(paths, configFile)
	}

	basePaths := []string{"/etc/cert-watcher/cert-watcher", "$XDG_CONFIG_HOME/cert-watcher", "$HOME/.config/cert-watcher", "./cert-watcher"}
	extensions := []string{"yaml", "yml"}
	for _, basePath := range basePaths {
		for _, extension := range extensions {
			paths = append(paths, basePath+"."+extension)
		}
	}

	return paths
}
