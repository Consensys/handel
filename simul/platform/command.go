package platform

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os/exec"
)

// Command is a wrapper around Go's Cmd that can also output the log on demand
type Command struct {
	*exec.Cmd
	pipe   io.Reader
	stdOut *bytes.Buffer
	stdErr *bytes.Buffer
}

// NewCommand returns a command that can outputs its stdout and stderr
func NewCommand(cmd string, args ...string) *Command {
	c := new(Command)
	c.stdOut = new(bytes.Buffer)
	c.stdErr = new(bytes.Buffer)
	c.Cmd = exec.Command(cmd, args...)

	outPipe, err := c.StdoutPipe()
	if err != nil {
		panic(err)
	}
	errPipe, err := c.StderrPipe()
	if err != nil {
		panic(err)
	}
	c.pipe = io.MultiReader(outPipe, errPipe)
	return c
}

// LineOutput continuously reads the stdout + stderr buffer and sends line by line
// output on the channel
func (c *Command) LineOutput() chan string {
	outCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(c.pipe)
		for scanner.Scan() {
			outCh <- scanner.Text()
		}
	}()
	return outCh
}

// ReadAll reads everything in the stdout + stderr reader
func (c *Command) ReadAll() string {
	buffOut, err := ioutil.ReadAll(c.pipe)
	if err != nil {
		panic("cant read output of command" + err.Error())
	}
	return string(buffOut)
}
