package aws

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	PemFile       string
	Regions       []string
	NbOfInstances int
	MasterTimeOut int
	SSHUser       string
}

func LoadConfig(path string) *Config {
	c := new(Config)
	_, err := toml.DecodeFile(path, c)
	if err != nil {
		panic(err)
	}
	return c
}
