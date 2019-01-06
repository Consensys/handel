package platform

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ConsenSys/handel/simul/lib"
	"github.com/ConsenSys/handel/simul/platform/aws"
)

type awsPlatform struct {
	aws           aws.Manager
	pemBytes      []byte
	master        aws.NodeController
	masterAddr    string
	monitorAddr   string
	monitorPort   int
	network       string
	allSlaveNodes []*aws.Instance
	masterCMDS    aws.MasterCommands
	slaveCMDS     aws.SlaveCommands
	cons          lib.Constructor
	awsConfig     *aws.Config
	resFile       string
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
	a.masterCMDS = aws.MasterCommands{CMDS}
	a.slaveCMDS = aws.SlaveCommands{CMDS}
	a.network = c.Network
	a.resFile = c.GetCSVFile()
	a.monitorPort = c.MonitorPort

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
	a.monitorAddr = aws.GenRemoteAddress(*masterInstance.PublicIP, c.MonitorPort)
	masterNode := lib.GenerateNode(cons, -1, masterAddr)
	nodeAndSync := aws.NodeAndSync{masterNode, ""}
	masterInstance.Nodes = []aws.NodeAndSync{nodeAndSync}

	//Create master controller
	master, err := aws.NewSSHNodeController(*masterInstance.PublicIP, a.pemBytes, a.awsConfig.SSHUser)
	if err != nil {
		return err
	}
	a.master = master

	for {
		err := master.Init()
		if err != nil {
			fmt.Println("Master Init failed, trying one more time", err, *masterInstance.ID, *masterInstance.PublicIP, *masterInstance.State)
			time.Sleep(3 * time.Second)
			continue
		}
		break
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
	a.master.Close()
	return a.aws.StopInstances()
}

func (a *awsPlatform) Start(idx int, r *lib.RunConfig) error {
	nodePerInstances := r.Nodes
	slaveNodes := a.allSlaveNodes[0:a.awsConfig.NbOfInstances]
	aws.UpdateInstances(slaveNodes, nodePerInstances, a.cons)

	writeRegFile(slaveNodes, a.masterCMDS.RegPath)
	//*** Start Master
	fmt.Println("[+] Registry file written to local storage(", r.Nodes, " nodes)")
	fmt.Println("[+] Transfering registry file to Master")
	a.master.CopyFiles(a.masterCMDS.RegPath)
	shareRegistryFile := a.masterCMDS.ShareRegistryFile()
	fmt.Println("[+] Master handel node:")
	for i := 0; i < len(shareRegistryFile); i++ {
		fmt.Println("       Exec:", i, shareRegistryFile[i])
		_, err := a.master.Run(shareRegistryFile[i])
		if err != nil {
			return err
		}
	}

	masterStart := a.masterCMDS.Start(
		a.masterAddr,
		a.awsConfig.NbOfInstances*nodePerInstances,
		a.awsConfig.MasterTimeOut,
		idx,
		r.Threshold,
		a.network,
		a.resFile,
		a.monitorPort)

	fmt.Println("       Exec:", len(shareRegistryFile)+1, masterStart)
	done := make(chan bool)
	go func() {
		_, err := a.master.Run(masterStart)
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

	for _, n := range inst.Nodes {
		startSlave := a.slaveCMDS.Start(a.masterAddr, n.Sync, a.monitorAddr, int(n.ID()), idx, n.Identity.Address())
		fmt.Println("Start Slave", startSlave)
		err := slaveController.Start(startSlave)
		if err != nil {
			panic(err)
		}
	}
	slaveController.Close()
}

func cmdToString(cmd []string) string {
	return strings.Join(cmd[:], " ")
}

func writeRegFile(instances []*aws.Instance, regPath string) {
	parser := lib.NewCSVParser()
	var nodes []*lib.Node
	for _, inst := range instances {
		for _, n := range inst.Nodes {
			nodes = append(nodes, n.Node)
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
