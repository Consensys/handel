package platform

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/platform/aws"
)

type awsPlatform struct {
	aws      aws.Manager
	pemBytes []byte
	//master        aws.NodeController
	masterAddr    string
	masterIP      string
	monitorAddr   string
	monitorPort   int
	network       string
	allSlaveNodes []*aws.Instance
	masterCMDS    aws.MasterCommands
	slaveCMDS     aws.SlaveCommands
	cons          lib.Constructor
	awsConfig     *aws.Config
	resFile       string
	c             *lib.Config
}

const s3Dir = "pegasysrndbucketvirginiav1"

// NewAws creates AWS Platform
func NewAws(aws aws.Manager, awsConfig *aws.Config) Platform {
	pemBytes, err := ioutil.ReadFile(awsConfig.PemFile)
	if err != nil {
		panic(err)
	}
	return &awsPlatform{aws: aws,
		pemBytes:  pemBytes,
		awsConfig: awsConfig,
	}
}

func (a *awsPlatform) pack(path string, c *lib.Config, binPath string) error {
	// Compile binaries
	//GOOS=linux GOARCH=amd64 go build
	os.Setenv("GOOS", a.awsConfig.TargetSystem)
	os.Setenv("GOARCH", a.awsConfig.TargetArch)
	cmd := NewCommand("go", "build", "-o", binPath, path)

	if err := cmd.Run(); err != nil {
		fmt.Println("stdout -> " + cmd.ReadAll())
		return err
	}
	return nil
}

func transferToS3(path string) error {
	fmt.Println("File", path)

	cmd := NewCommand("aws", "s3", "cp", path, "s3://"+s3Dir+"/tmp/")
	if err := cmd.Run(); err != nil {
		fmt.Println("stdout -> " + cmd.ReadAll())
		return err
	}
	return nil
}

func (a *awsPlatform) Configure(c *lib.Config) error {

	CMDS := aws.NewCommands(
		"/tmp/masterAWS",
		"/tmp/nodeAWS",
		"/tmp/aws.conf",
		"/tmp/aws.csv",
		"https://s3.amazonaws.com/"+s3Dir)

	a.masterCMDS = aws.MasterCommands{Commands: CMDS}
	a.slaveCMDS = aws.SlaveCommands{Commands: CMDS, SameBinary: true, SyncBasePort: 6000}
	a.network = c.Network
	a.resFile = c.GetCSVFile()
	a.monitorPort = c.MonitorPort
	a.c = c

	// Compile binaries
	a.pack("github.com/ConsenSys/handel/simul/node", c, CMDS.SlaveBinPath)
	a.pack("github.com/ConsenSys/handel/simul/master", c, CMDS.MasterBinPath)

	// write config
	if err := c.WriteTo(CMDS.ConfPath); err != nil {
		return err
	}

	//Start EC2 instances
	if err := a.aws.StartInstances(); err != nil {
		return err
	}

	// Create master and slave instances
	masterInstance, slaveInstances, err := makeMasterAndSlaves(a.aws.Instances())
	if err != nil {
		fmt.Println(err)
		return err
	}

	cons := c.NewConstructor()
	a.cons = cons
	masterAddr := aws.GenRemoteAddress(*masterInstance.PublicIP, 5000)
	a.masterAddr = masterAddr
	a.masterIP = *masterInstance.PublicIP
	a.monitorAddr = aws.GenRemoteAddress(*masterInstance.PublicIP, c.MonitorPort)
	masterNode := lib.GenerateNode(cons, -1, masterAddr)
	masterInstance.Nodes = []*lib.Node{masterNode}
	//Create master controller
	master, err := a.connectToMaster()
	if err != nil {
		return err
	}
	master.Run(a.masterCMDS.Kill(), nil)

	fmt.Println("[+] Master Instances")
	fmt.Println("	 [-] Instance ", *masterInstance.ID, *masterInstance.State, masterAddr)
	fmt.Println()
	fmt.Println("[+] Avaliable Slave Instances:")

	for i, inst := range slaveInstances {
		fmt.Println("	 [-] Instance ", i, *inst.ID, *inst.State, *inst.PublicIP)
	}

	fmt.Println("[+] Transfering files to S3:", CMDS.MasterBinPath, CMDS.SlaveBinPath, CMDS.ConfPath)

	transferToS3(CMDS.MasterBinPath)
	transferToS3(CMDS.SlaveBinPath)
	transferToS3(CMDS.ConfPath)

	configure := a.masterCMDS.Configure()

	//*** Configure Master
	fmt.Println("[+] Configuring Master")
	for idx := 0; idx < len(configure); idx++ {
		fmt.Println("       Exec:", idx, configure[idx])
		err := master.Run(configure[idx], nil)
		if err != nil {
			return err
		}
	}

	//*** Configure Slaves
	slaveCmds := a.slaveCMDS.Configure(*masterInstance.PublicIP)

	fmt.Println("")
	fmt.Println("")
	fmt.Println("[+] Configuring Slaves:")

	var wg sync.WaitGroup

	var counter int32
	for _, slave := range slaveInstances {
		wg.Add(1)
		// TODO This might become a problem for large number of slaves,
		// limit number of go-routines running concurrently if this is the case

		go func(slave aws.Instance) {
			slaveNodeController, err := aws.NewSSHNodeController(*slave.PublicIP, a.pemBytes, a.awsConfig.SSHUser)
			if err != nil {
				panic(err)
			}
			fmt.Println("    - Configuring Slave", *slave.PublicIP)
			if err := configureSlave(slaveNodeController, slaveCmds, a.slaveCMDS.Kill()); err != nil {
				fmt.Println("  Problem with Slave", *slave.PublicIP, err)
				panic(err)
			}
			atomic.AddInt32(&counter, 1)
			counterValue := atomic.LoadInt32(&counter)
			fmt.Println("    - Configuring Slave Done", counterValue, *slave.PublicIP)
			wg.Done()
		}(*slave)
		a.allSlaveNodes = append(a.allSlaveNodes, slave)
	}
	wg.Wait()
	master.Close()
	return nil
}

func configureSlave(slaveNodeController aws.NodeController, slaveCmds map[int]string, kill string) error {
	if err := slaveNodeController.Init(); err != nil {
		return err
	}
	defer slaveNodeController.Close()
	//slaveNodeController.Run(kill, nil)

	for idx := 0; idx < len(slaveCmds); idx++ {
		err := slaveNodeController.Run(slaveCmds[idx], nil)
		if err != nil {
			fmt.Println("Error:", slaveCmds[idx])
			return err
		}
	}
	return nil
}

func (a *awsPlatform) Cleanup() error {
	//return a.aws.StopInstances()
	return nil
}

func (a *awsPlatform) Start(idx int, r *lib.RunConfig) error {
	fmt.Println("Start run", idx)
	//Create master controller
	master, err := a.connectToMaster()
	if err != nil {
		panic(err)
	}

	slaveNodes := a.allSlaveNodes[0:r.Processes]
	allocator := a.c.NewAllocator()
	ids := allocator.Allocate(r.Nodes, r.Failing)
	aws.UpdateInstances(ids, r.Nodes, slaveNodes, a.cons)
	writeRegFile(r.Nodes, slaveNodes, a.masterCMDS.RegPath)
	//*** Start Master
	fmt.Println("[+] Registry file written to local storage(", r.Nodes, " nodes)")
	fmt.Println("[*] Transferring registry file to S3")
	transferToS3(a.masterCMDS.RegPath)

	masterStart := a.masterCMDS.Start(
		a.masterAddr,
		a.awsConfig.MasterTimeOut,
		idx,
		a.network,
		a.resFile,
		a.monitorPort,
	)

	fmt.Println("       Exec:", masterStart)
	done := make(chan bool)
	go func() {
		master.Run(a.masterCMDS.Kill(), nil)
		err = master.Run(masterStart, nil)
		if err != nil {
			panic(err)
		}

		done <- true
	}()
	//*** Start slaves
	var wg sync.WaitGroup
	for _, n := range slaveNodes {
		wg.Add(1)
		go func(slaveNode aws.Instance) {
			// TODO This might become a problem for large number of slaves,
			// limit numebr of go-routines running concurrently if this is the case
			a.startSlave(slaveNode, idx)
			wg.Done()
		}(*n)
	}
	wg.Wait()
	<-done
	master.Close()
	return nil
}

func (a *awsPlatform) startSlave(inst aws.Instance, idx int) {
	slaveController, err := aws.NewSSHNodeController(*inst.PublicIP, a.pemBytes, a.awsConfig.SSHUser)
	if err != nil {
		panic(err)
	}

	cpyFiles := a.slaveCMDS.CopyRegistryFileFromSharedDirToLocalStorage()
	if err := slaveController.Init(); err != nil {
		panic(err)
	}

	for i := 0; i < len(cpyFiles); i++ {
		fmt.Println(*inst.PublicIP, cpyFiles[i])
		if err := slaveController.Run(cpyFiles[i], nil); err != nil {
			panic(err)
		}
	}

	cmd := a.slaveCMDS.Start(a.masterAddr, a.monitorAddr, inst, idx)
	fmt.Println("Start Slave", cmd)
	pr, pw := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pr)
		for {
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					panic(err)
				}
				break
			}
			fmt.Println(scanner.Text())
		}
	}()

	slaveController.Run(a.slaveCMDS.Kill(), nil)

	err = slaveController.Run(cmd, pw)
	if err != nil {
		fmt.Println("Error "+*inst.PublicIP, err)
		panic(err)
	}
	slaveController.Close()
}

func (a *awsPlatform) connectToMaster() (aws.NodeController, error) {
	//Create master controller
	master, err := aws.NewSSHNodeController(a.masterIP, a.pemBytes, a.awsConfig.SSHUser)
	if err != nil {
		return nil, err
	}

	for {
		err := master.Init()
		if err != nil {
			fmt.Println("Master Init failed, trying one more time", err)
			time.Sleep(3 * time.Second)
			continue
		}
		break
	}
	return master, nil
}

func cmdToString(cmd []string) string {
	return strings.Join(cmd[:], " ")
}

func writeRegFile(total int, instances []*aws.Instance, regPath string) {
	parser := lib.NewCSVParser()
	var nodes = make([]*lib.Node, total)
	for _, inst := range instances {
		for _, n := range inst.Nodes {
			nodes[int(n.ID())] = n
		}
	}
	lib.WriteAll(nodes, parser, regPath)
}

func makeMasterAndSlaves(allAwsInstances []aws.Instance) (*aws.Instance, []*aws.Instance, error) {
	var masterInstance aws.Instance
	var slaveInstances []*aws.Instance
	nbOfMasterIns := 0

	for _, inst := range allAwsInstances {
		if inst.Tag == aws.RnDMasterTag {
			if nbOfMasterIns > 1 {
				return nil, nil, errors.New("more than one Master instance available")
			}
			masterInstance = inst
			nbOfMasterIns++
		} else {
			si := inst
			slaveInstances = append(slaveInstances, &si)
		}
	}
	return &masterInstance, slaveInstances, nil
}
