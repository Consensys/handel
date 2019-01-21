package aws

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type sshController struct {
	client  *ssh.Client
	sshHost string
	config  *ssh.ClientConfig
}

// NewSSHNodeController creates ssh based NodeController
func NewSSHNodeController(sshAddr string, pemBytes []byte, user string) (NodeController, error) {
	sshHost := net.JoinHostPort(sshAddr, "22") //sshHostAddr(node.Address())

	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return &sshController{sshHost: sshHost, config: config}, nil
}

//func (sshCMD *sshController) Addr() string {
//	return sshCMD.node
//}

func (sshCMD *sshController) Init() error {
	conn, err := ssh.Dial("tcp", sshCMD.sshHost, sshCMD.config)
	if err != nil {
		return err
	}
	sshCMD.client = conn
	return nil
}

//CopyFiles copies files from local to remote host using sftp
func (sshCMD *sshController) CopyFiles(files ...string) error {
	// create new SFTP client
	sftpClient, err := sftp.NewClient(sshCMD.client)
	if err != nil {
		return err
	}
	//defer sftpClient.Close()
	for _, file := range files {
		copyFile(sftpClient, file)
	}
	return nil
}

func copyFile(sftpClient *sftp.Client, file string) error {
	// create destination file
	dstFile, err := sftpClient.Create(file)

	if err != nil {
		return err
	}
	defer dstFile.Close()

	// create source file
	srcFile, err := os.Open(file)
	if err != nil {
		return err
	}

	// copy source file to destination file
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

//Run runs command on a remote host using ssh and waits for output
func (sshCMD *sshController) Run(command string, pw *io.PipeWriter) error {
	session, err := sshCMD.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// +++ We need to be careful here:
	// From the doc:
	// If the StdoutPipe reader is
	// not serviced fast enough it may eventually cause the
	// remote command to block.
	if pw != nil {
		outPipe, err := session.StdoutPipe()
		if err != nil {
			return err
		}
		errPipe, err := session.StderrPipe()
		if err != nil {
			return err
		}
		pipe := io.MultiReader(outPipe, errPipe)
		go func() {
			io.Copy(pw, pipe)
		}()
	}
	err = session.Run(command)
	if err != nil {
		fmt.Println("Warning SSH: ", command, sshCMD.sshHost, err)
		return err
	}
	return nil
}

func (sshCMD *sshController) Start(command string) error {
	session, err := sshCMD.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	err = session.Start(command)
	if err != nil {
		fmt.Println("Warning SSH: ", command, sshCMD.sshHost, err)
		return err
	}
	return nil
}

//Close closes ssh session
func (sshCMD *sshController) Close() {
	sshCMD.client.Close()
}

func sshHostAddr(addr string) (string, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	newAddr := net.JoinHostPort(host, "22")
	return newAddr, nil
}
