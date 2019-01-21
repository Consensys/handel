package aws

import "io"

// NodeController represents avaliable operations to perform on a remote node
type NodeController interface {
	// CopyFiles copies files to equivalent location on a remote host
	// for example "/tmp/aws.csv" from localhost will be placed in
	// "/tmp/aws.csv" on the remote host
	CopyFiles(files ...string) error
	// Run runs command on a remote node, for example Run("ls -l") and blocks until completion
	Run(command string, pw *io.PipeWriter) error
	// Run starts command on a remote node
	Start(command string) error

	// Init inits connection to the remote node
	Init() error
	// Close
	Close()
}
