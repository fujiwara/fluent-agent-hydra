package hydra

import (
	"github.com/BurntSushi/toml"
	"log"
)

type Config struct {
	TagPrefix string
	FieldName string
	Servers   []string
	Logs      []logfile
}

type logfile struct {
	Tag  string
	File string
}

func ReadConfig(filename string) (Config, error) {
	var config Config
	log.Println("[info] Loading config file:", filename)
	if _, err := toml.DecodeFile(filename, &config); err != nil {
		return config, err
	}
	return config, nil
}
