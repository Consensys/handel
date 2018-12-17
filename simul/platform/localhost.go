package platform

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ConsenSys/handel/simul/lib"
)

type localPlatform struct {
	c        *lib.Config
	regPath  string
	binPath  string
	confPath string
	sync.Mutex
	cmds []*Command
}

// NewLocalhost returns a Platform that is executing binaries on localhost
func NewLocalhost() Platform { return &localPlatform{} }

func (l *localPlatform) Configure(c *lib.Config) error {
	l.c = c
	l.regPath = "/tmp/local.csv"
	l.binPath = "/tmp/local.bin"
	l.confPath = "/tmp/local.conf"
	// Compile binaries
	pack := "github.com/ConsenSys/handel/simul/node"
	cmd := NewCommand("go", "build", "-o", l.binPath, pack)
	if err := cmd.Run(); err != nil {
		fmt.Println("command output -> " + cmd.ReadAll())
		return err
	}
	// write config
	if err := c.WriteTo(l.confPath); err != nil {
		return err
	}
	return nil
}

func (l *localPlatform) Cleanup() error {
	os.RemoveAll(l.regPath)
	l.Lock()
	defer l.Unlock()

	for _, c := range l.cmds {
		if err := c.Process.Kill(); err != nil {
			//fmt.Printf("[-] error killing command %d: %s\n", i, err)
		}
	}
	return nil
}

func (l *localPlatform) Start(idx int, r *lib.RunConfig) error {
	// 1. Generate & write the registry file
	cons := l.c.NewConstructor()
	parser := lib.NewCSVParser()
	addresses, syncs := genLocalAddresses(r.Nodes)
	nodes := lib.GenerateNodes(cons, addresses)
	lib.WriteAll(nodes, parser, l.regPath)
	fmt.Println("[+] Registry file written (", r.Nodes, " nodes)")

	// 2. Run the sync master
	masterAddr := lib.FindFreeUDPAddress()
	master := lib.NewSyncMaster(masterAddr, r.Nodes)
	fmt.Println("[+] Master synchronization daemon launched")

	// 3. Run binaries
	commands := make([]*Command, r.Nodes)
	doneCh := make(chan int, r.Nodes)
	errCh := make(chan int, r.Nodes)
	sameArgs := []string{"-config", l.confPath,
		"-registry", l.regPath,
		"-master", masterAddr,
		"-monitor", l.c.GetMonitorAddress("127.0.0.1")}

	for i := 0; i < r.Nodes; i++ {
		// 3.1 prepare args
		args := make([]string, len(sameArgs))
		copy(args, sameArgs)
		args = append(args, []string{"-id", strconv.Itoa(i),
			"-sync", syncs[i],
			"-run", strconv.Itoa(idx)}...)

		// 3.2 run command
		commands[i] = NewCommand(l.binPath, args...)
		go func(j int) {
			fmt.Printf("[+] Starting node %d.\n", j)
			if err := commands[j].Start(); err != nil {
				fmt.Printf("node %d: %s\n",
					j, commands[j].ReadAll())
				errCh <- j
				return
			}

			go func() {
				for str := range commands[j].LineOutput() {
					fmt.Printf("NODE %d: %s\n", j, str)
				}
			}()
			time.Sleep(200 * time.Millisecond)
			if err := commands[j].Wait(); err != nil {
				fmt.Printf("node %d: %s\n", j, commands[j].ReadAll())

				errCh <- j
			}
			doneCh <- j
		}(i)
	}

	l.Lock()
	l.cmds = commands
	l.Unlock()

	// 4. Wait for the master to have synced up every node
	select {
	case <-master.WaitAll():
		master.Reset()
		fmt.Printf("[+] Master full synchronization done.\n")
	case <-time.After(5 * time.Minute):
		panic("timeout after 2 mn")
	}

	// 5. Wait all finished - then tell them to quit
	select {
	case <-master.WaitAll():
		fmt.Printf("[+] Master - finished synchronization done.\n")
	case <-time.After(l.c.GetMaxTimeout()):
		panic(fmt.Sprintf("timeout after %s", l.c.GetMaxTimeout()))
	}

	// 6. Wait for all binaries to finish - clean finishing
	maxTimeout := make(chan bool, 1)
	go func() { <-time.After(l.c.GetMaxTimeout()); maxTimeout <- true }()
	var nOk, nErr int
	for {
		select {
		case <-doneCh:
			nOk++
		case <-errCh:
			nErr++
		case <-maxTimeout:
			panic("global timeout reached")
		}
		if nOk+nErr >= r.Nodes {
			fmt.Printf("nOk = %d, nErr = %d\n", nOk, nErr)
			break
		}
	}

	fmt.Printf("[+] Localhost round %d finished - success !\n", idx)
	/*for i, command := range commands {*/
	//if str := command.Stdout(); str != "" {
	//fmt.Printf(" ----- node %d output -----\n\t%s\n ----------------\n", i, str)
	//}

	/*}*/
	return nil
}

// this generates n * 2 addresses: one for handel, one for the sync
func genLocalAddresses(n int) ([]string, []string) {
	var addresses = make([]string, 0, n)
	var syncs = make([]string, 0, n)
	base := 3000
	for i := 0; i < n; i++ {
		port1 := base + i*2
		port2 := port1 + 1
		addr1 := "127.0.0.1:" + strconv.Itoa(port1)
		addr2 := "127.0.0.1:" + strconv.Itoa(port2)
		addresses = append(addresses, addr1)
		syncs = append(syncs, addr2)
	}
	return addresses, syncs
}
