package lib

import (
	"encoding/hex"

	"github.com/ConsenSys/handel"
)

// NodeRecord holds a node's information in a readable string format
type NodeRecord struct {
	ID          int32
	Addr        string
	Private     string // hex encoded
	Public      string // hex encoded
	IsByzantine bool
}

// Node is similar to a NodeRecord but decoded
type Node struct {
	SecretKey
	handel.Identity
	Active      bool
	IsByzantine bool
}

// ToRecord maps a Node to a NodeRecord, its string-human-readable equivalent
func (n *Node) ToRecord() (*NodeRecord, error) {
	nr := new(NodeRecord)
	nr.ID = n.ID()
	nr.Addr = n.Address()
	buff, err := n.SecretKey.MarshalBinary()
	if err != nil {
		return nil, err
	}
	nr.Private = hex.EncodeToString(buff)
	buff, err = n.Identity.PublicKey().(PublicKey).MarshalBinary()
	if err != nil {
		return nil, err
	}
	nr.Public = hex.EncodeToString(buff)
	nr.IsByzantine = n.IsByzantine
	return nr, nil
}

// ToNode the private and public key from the given constructor and returns the
// secret key and the corresponding identity
func (n *NodeRecord) ToNode(c Constructor) (*Node, error) {
	buff, err := hex.DecodeString(n.Private)
	if err != nil {
		return nil, err
	}
	sk := c.SecretKey()
	if err := sk.UnmarshalBinary(buff); err != nil {
		return nil, err
	}

	buff, err = hex.DecodeString(n.Public)
	if err != nil {
		return nil, err
	}
	pk := c.PublicKey().(PublicKey)
	if err = pk.UnmarshalBinary(buff); err != nil {
		return nil, err
	}
	identity := handel.NewStaticIdentity(int32(n.ID), n.Addr, pk)
	return &Node{SecretKey: sk, Identity: identity, IsByzantine: n.IsByzantine}, nil
}
