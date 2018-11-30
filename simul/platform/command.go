package platform

import (
	"bytes"
	"io"
	"io/ioutil"
	"os/exec"
)

// Command is a wrapper around Go's Cmd that can also output the log on demand
type Command struct {
	*exec.Cmd
	stdOut *bytes.Buffer
	stdErr *bytes.Buffer
}

// NewCommand returns a command that can outputs its stdout and stderr
func NewCommand(cmd string, args ...string) *Command {
	c := new(Command)
	c.stdOut = new(bytes.Buffer)
	c.stdErr = new(bytes.Buffer)
	c.Cmd = exec.Command(cmd, args...)
	c.Cmd.Stdout = c.stdOut
	c.Cmd.Stderr = c.stdErr
	return c
}

// Stdout returns the standard output as a string
func (c *Command) Stdout() string {
	return c.read(c.stdOut)
}

func (c *Command) read(r io.Reader) string {
	buffOut, err := ioutil.ReadAll(r)
	if err != nil {
		panic("cant read output of command" + err.Error())
	}
	return string(buffOut)
}

// Stderr returns the standard error  as a string
func (c *Command) Stderr() string {
	return c.read(c.stdErr)
}
