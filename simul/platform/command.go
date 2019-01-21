package platform

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os/exec"
	"sync"
)

// Command is a wrapper around Go's Cmd that can also output the log on demand
type Command struct {
	*exec.Cmd
	pipe   io.Reader
	out    *bytes.Buffer
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// NewCommand returns a command that can outputs its stdout and stderr
func NewCommand(cmd string, args ...string) *Command {
	c := new(Command)
	c.Cmd = exec.Command(cmd, args...)

	var err error
	c.stdout, err = c.StdoutPipe()
	if err != nil {
		panic(err)
	}
	c.stderr, err = c.StderrPipe()
	if err != nil {
		panic(err)
	}
	return c
}

// LineOutput continuously reads the stdout + stderr buffer and sends line by line
// output on the channel
func (c *Command) LineOutput() chan string {
	outCh := make(chan string, 100)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		c.readAndRedirect(c.stdout, outCh)
		wg.Done()
	}()
	go func() {
		c.readAndRedirect(c.stderr, outCh)
		wg.Done()
	}()
	go func() { wg.Wait(); close(outCh) }()
	return outCh
}

func (c *Command) readAndRedirect(r io.Reader, ch chan string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		ch <- scanner.Text()
	}
	return
}

// ReadAll returns stdout read entirely
func (c *Command) ReadAll() string {
	buffOut, err := ioutil.ReadAll(c.stdout)
	if err != nil {
		panic("cant read output of command" + err.Error())
	}
	return string(buffOut)
}
