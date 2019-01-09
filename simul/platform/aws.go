package platform

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
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

func (a *awsPlatform) Configure(c *lib.Config) error {

	CMDS := aws.NewCommands("/tmp/masterAWS", "/tmp/nodeAWS", "/tmp/aws.conf", "/tmp/aws.csv")
	a.masterCMDS = aws.MasterCommands{Commands: CMDS}
	a.slaveCMDS = aws.SlaveCommands{Commands: CMDS}
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

	fmt.Println("[+] Master Instances")
	fmt.Println("	 [-] Instance ", *masterInstance.ID, *masterInstance.State, masterAddr)
	fmt.Println()
	fmt.Println("[+] Avaliable Slave Instances:")

	for i, inst := range slaveInstances {
		fmt.Println("	 [-] Instance ", i, *inst.ID, *inst.State, *inst.PublicIP)
	}

	fmt.Println("[+] Transfering files to Master:", CMDS.MasterBinPath, CMDS.SlaveBinPath, CMDS.ConfPath)
	master.CopyFiles(CMDS.MasterBinPath, CMDS.SlaveBinPath, CMDS.ConfPath)
	configure := a.masterCMDS.Configure()

	//*** Configure Master
	fmt.Println("[+] Configuring Master")
	for idx := 0; idx < len(configure); idx++ {
		fmt.Println("       Exec:", idx, configure[idx])
		_, err := master.Run(configure[idx])
		if err != nil {
			return err
		}
	}

	//*** Configure Slaves
	slaveCmds := a.slaveCMDS.Configure(*masterInstance.PublicIP)

	fmt.Println("")
	fmt.Println("")
	fmt.Println("[+] Configuring Slaves:")

	//addresses, syncs := aws.GenRemoteAddresses(slaveInstances)
	var wg sync.WaitGroup

	for _, slave := range slaveInstances {
		//	node := lib.GenerateNode(cons, i, addr)
		//	nodeAndSync := aws.NodeAndSync{node, syncs[i]}
		wg.Add(1)
		// TODO This might become a problem for large number of slaves,
		// limit numebr of go-routines running concurrently if this is the case

		go func(slave aws.Instance) {

			slaveNodeController, err := aws.NewSSHNodeController(*slave.PublicIP, a.pemBytes, a.awsConfig.SSHUser)
			if err != nil {
				panic(err)
			}
			configureSlave(slaveNodeController, slaveCmds)
			fmt.Println("    - Slave", *slave.PublicIP)
			wg.Done()
		}(*slave)
		a.allSlaveNodes = append(a.allSlaveNodes, slave)
	}
	wg.Wait()
	master.Close()
	return nil
}

func configureSlave(slaveNodeController aws.NodeController, slaveCmds map[int]string) error {
	slaveNodeController.Init()
	defer slaveNodeController.Close()

	for idx := 0; idx < len(slaveCmds); idx++ {
		_, err := slaveNodeController.Run(slaveCmds[idx])
		if err != nil {
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

	//Create master controller
	master, err := a.connectToMaster()
	if err != nil {
		return nil
	}

	slaveNodes := a.allSlaveNodes[0:a.awsConfig.NbOfInstances]
	allocator := a.c.NewAllocator()
	ids := allocator.Allocate(r.Nodes, r.Failing)
	aws.UpdateInstances(ids, r.Nodes, slaveNodes, a.cons)
	writeRegFile(r.Nodes, slaveNodes, a.masterCMDS.RegPath)
	//*** Start Master
	fmt.Println("[+] Registry file written to local storage(", r.Nodes, " nodes)")
	fmt.Println("[+] Transfering registry file to Master")
	master.CopyFiles(a.masterCMDS.RegPath)
	shareRegistryFile := a.masterCMDS.ShareRegistryFile()
	fmt.Println("[+] Master handel node:")
	for i := 0; i < len(shareRegistryFile); i++ {
		fmt.Println("       Exec:", i, shareRegistryFile[i])
		_, err := master.Run(shareRegistryFile[i])
		if err != nil {
			panic(err)
		}
	}

	masterStart := a.masterCMDS.Start(
		a.masterAddr,
		//a.awsConfig.NbOfInstances*nodePerInstances,
		r.Nodes,
		r.Failing,
		a.awsConfig.NbOfInstances,
		a.awsConfig.MasterTimeOut,
		idx,
		r.Threshold,
		a.network,
		a.resFile,
		a.monitorPort)

	fmt.Println("       Exec:", len(shareRegistryFile)+1, masterStart)
	done := make(chan bool)
	go func() {
		_, err := master.Run(masterStart)
		if err != nil {
			panic(err)
		}

		done <- true
	}()
	//*** Starte slaves
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
	cpyFiles := a.slaveCMDS.CopyRegistryFileFromSharedDirToLocalStorage()
	slaveController, err := aws.NewSSHNodeController(*inst.PublicIP, a.pemBytes, a.awsConfig.SSHUser)

	if err != nil {
		panic(err)
	}
	if err := slaveController.Init(); err != nil {
		panic(err)
	}

	for i := 0; i < len(cpyFiles); i++ {
		_, err := slaveController.Run(cpyFiles[i])
		if err != nil {
			panic(err)
		}
	}

	idArgs := []string{}

	for _, n := range inst.Nodes {
		if !n.Active {
			continue
		}
		id := " -id " + strconv.Itoa(int(n.ID()))
		idArgs = append(idArgs, id)
	}

	ids := strings.Join(idArgs, "")
	startSlave := a.slaveCMDS.Start(a.masterAddr, inst.Sync, a.monitorAddr, ids, idx, "log.txt")
	fmt.Println("Start Slave", startSlave)
	err = slaveController.Start(startSlave)
	if err != nil {
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
				return nil, nil, errors.New("More than one Master instance avaliable")
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
