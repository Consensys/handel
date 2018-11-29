package utils

import (
	"fmt"
	"strconv"
)

type noPubKeyParser struct {
}

// NewEmptyPublicKeyCsvParser skips parsing public key
// This is used only in transport example, once we agree on PubKey
// serialization format another parser will be provided
func NewEmptyPublicKeyCsvParser() CsvConstructor {
	return &noPubKeyParser{}
}

// Read implements CsvConstructor
func (lp *noPubKeyParser) Read(line []string) (*NodeRecord, error) {
	i, err := strconv.ParseInt(line[0], 10, 32)
	if err != nil {
		return nil, err
	}
	id := int32(i)

	p, err := strconv.ParseInt(line[1], 10, 32)
	if err != nil {
		return nil, err
	}
	pt := int(p)
	addr := fmt.Sprintf("%s:%d", line[2], pt)

	//TODO read peer's serialized public key
	nodeRecord := &NodeRecord{id: id, port: pt, addr: addr, pubKey: ""}
	return nodeRecord, nil
}
