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
	targetSystem  string
	targetArch    string
	user          string
	pemBytes      []byte
	master        aws.NodeController
	masterAddr    string
	allSlaveNodes []*aws.Instance
	masterCMDS    aws.MasterCommands
	slaveCMDS     aws.SlaveCommands
}

//TODO this options should be placed in separate config
const masterTimeOut = 4
const nodePerInstances = 128

// cross-compilation option
const targetSystem = "linux"
const targetArch = "amd64"
const user = "ubuntu"

// NewAws creates AWS Platform
func NewAws(aws aws.Manager, pemFile string) Platform {
	pemBytes, err := ioutil.ReadFile(pemFile)
	if err != nil {
		panic(err)
	}
	return &awsPlatform{aws: aws,
		targetSystem: targetSystem,
		targetArch:   targetArch,
		user:         user,
		pemBytes:     pemBytes,
	}
}

func (a *awsPlatform) pack(path string, c *lib.Config, binPath string) error {
	// Compile binaries
	//GOOS=linux GOARCH=amd64 go build
	os.Setenv("GOOS", a.targetSystem)
	os.Setenv("GOARCH", a.targetArch)
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
	masterAddr := aws.GenRemoteAddress(*masterInstance.PublicIP, 5000)
	a.masterAddr = masterAddr
	masterNode := lib.GenerateNode(cons, -1, masterAddr)
	nodeAndSync := aws.NodeAndSync{masterNode, ""}
	masterInstance.Nodes = []aws.NodeAndSync{nodeAndSync}

	//Create master controller
	master, err := aws.NewSSHNodeController(*masterInstance.PublicIP, a.pemBytes, a.user)
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
	//master.CopyFiles(CMDS.MasterBinPath, CMDS.SlaveBinPath, CMDS.ConfPath)
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
	fmt.Println(*masterInstance.PublicIP)
	slaveCmds := a.slaveCMDS.Configure(*masterInstance.PublicIP)
	fmt.Println(*masterInstance.PublicIP)

	fmt.Println("")
	fmt.Println("")
	fmt.Println("[+] Configuring Slaves:")

	aws.UpdateInstances(slaveInstances, nodePerInstances, cons)
	//addresses, syncs := aws.GenRemoteAddresses(slaveInstances)
	var wg sync.WaitGroup

	for _, slave := range slaveInstances {
		//	node := lib.GenerateNode(cons, i, addr)
		//	nodeAndSync := aws.NodeAndSync{node, syncs[i]}
		wg.Add(1)
		// TODO This might become a problem for large number of slaves,
		// limit numebr of go-routines running concurrently if this is the case

		go func(slave aws.Instance) {

			slaveNodeController, err := aws.NewSSHNodeController(*slave.PublicIP, a.pemBytes, a.user)
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
	//a.master.Close()
	return nil //a.aws.StopInstances()
}

func (a *awsPlatform) Start(idx int, r *lib.RunConfig) error {

	/*
		nbOfInstances := len(a.allSlaveNodes)
		if r.Nodes > nbOfInstances {
			msg := fmt.Sprintf(`Not enough EC2 instances, number of nodes to sart: %d
		               , number of avaliable EC2 instances: %d`, r.Nodes, nbOfInstances)
			return errors.New(msg)
		}*/
	slaveNodes := a.allSlaveNodes[0:r.Nodes]

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

	masterStart := a.masterCMDS.Start(a.masterAddr, r.Nodes*nodePerInstances, masterTimeOut)
	fmt.Println("       Exec:", len(shareRegistryFile)+1, masterStart)
	a.master.Start(masterStart)

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
	return nil
}

func (a *awsPlatform) startSlave(inst aws.Instance, idx int) {
	cpyFiles := a.slaveCMDS.CopyRegistryFileFromSharedDirToLocalStorage()
	slaveController, err := aws.NewSSHNodeController(*inst.PublicIP, a.pemBytes, a.user)

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
		startSlave := a.slaveCMDS.Start(a.masterAddr, n.Sync, int(n.ID()), idx, n.Identity.Address())
		fmt.Println("Start Slave", startSlave)
		slaveController.Run(startSlave)
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
