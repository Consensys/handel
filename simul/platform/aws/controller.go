package aws

import "github.com/ConsenSys/handel/simul/lib"

// NodeController represents avaliable operations to perform on a remote node
type NodeController interface {
	// CopyFiles copies files to equivalent location on a remote host
	// for example "/tmp/aws.csv" from localhost will be placed in
	// "/tmp/aws.csv" on the remote host
	CopyFiles(files ...string) error
	// Node returns underlying NodeAndSync
	//	Node() NodeAndSync
	// Run runs command on a remote node, for example Run("ls -l") and blocks until completion
	Run(command string) (string, error)
	// Start runs command on a remote node, doesn't block
	Start(command string) error
	// Init inits connection to the remote node
	Init() error
	// Close
	Close()
}

// NodeAndSync cpmbines Node and Sync address
type NodeAndSync struct {
	*lib.Node
	Sync string
}
