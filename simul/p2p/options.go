package p2p

import (
	"strconv"
)

// Opts represents generic string like options given in the TOML-encoded config
type Opts map[string]string

func (o *Opts) String(k string) (string, bool) {
	s, e := (*o)[k]
	return s, e
}

// Int returns the value stored at the given key converted to an int
func (o *Opts) Int(k string) (int, bool) {
	s, e := (*o)[k]
	if !e {
		return 0, false
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return i, true

}
