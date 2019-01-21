package aws

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	PemFile       string
	Regions       []string
	MasterTimeOut int
	SSHUser       string
	TargetSystem  string
	TargetArch    string
	CopyBinFiles  bool
	ConfTimeout   int
}

func LoadConfig(path string) *Config {
	c := new(Config)
	_, err := toml.DecodeFile(path, c)
	if err != nil {
		panic(err)
	}
	return c
}
