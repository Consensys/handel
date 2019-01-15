package aws

import (
	"strconv"
	"strings"
	//"github.com/ConsenSys/handel/simul/platform/aws"
)

// Commands represents AWS platform specyfic commands.
// Master instance creates NFS server and shared directory.
// When needed, Slave instances copy appropriate files form the shared directory
// to they local file system.
type Commands struct {
	MasterBinPath string
	SlaveBinPath  string
	ConfPath      string
	RegPath       string
}

// MasterCommands commands invoked on a master node
type MasterCommands struct {
	Commands
}

//SlaveCommands commands invoked on a slave node
type SlaveCommands struct {
	Commands
	SameBinary   bool
	SyncBasePort int
}

const logFile = "log"
const sharedDir = "$HOME/sharedDir"

// NewCommands creates an instance of Commands
func NewCommands(masterBinPath, slaveBinPath, confPath, regPath string) Commands {
	return Commands{
		MasterBinPath: masterBinPath,
		SlaveBinPath:  slaveBinPath,
		ConfPath:      confPath,
		RegPath:       regPath,
	}
}

// Configure configures EC2 master instance:
// - intsalls the NFS server
// - exports "shared" directory
// - copies apripirate files to that directory
func (c MasterCommands) Configure() map[int]string {
	cmds := make(map[int]string)
	cmds[0] = "sudo apt-get install nfs-kernel-server"
	cmds[1] = "sudo service nfs-kernel-server start"
	cmds[2] = "mkdir -p " + sharedDir
	cmds[3] = "sudo chmod 777 /etc/exports"
	//	cmds[4] = "cat /etc/exports" // *(rw,no_subtree_check,no_root_squash,sync,insecure) > /etc/exports"
	cmds[4] = "cp " + c.MasterBinPath + " " + sharedDir
	cmds[5] = "cp " + c.SlaveBinPath + " " + sharedDir
	cmds[6] = "cp " + c.ConfPath + " " + sharedDir
	cmds[7] = "sudo service nfs-kernel-server reload"
	return cmds
}

// ShareRegistryFile copies registry file to the shared directory
func (c MasterCommands) ShareRegistryFile() map[int]string {
	cmds := make(map[int]string)
	cmds[0] = "cp " + c.RegPath + " " + sharedDir
	cmds[1] = "chmod 777 " + c.MasterBinPath
	return cmds
}

// Start starts master executable
func (c MasterCommands) Start(masterAddr string, timeOut int, run int, network, resFile string, monitorPort int) string {
	return "nohup " + c.MasterBinPath + " -masterAddr " + masterAddr + " -timeOut " + strconv.Itoa(timeOut) + " -run " + strconv.Itoa(run) + " -network " + network + " -resultFile " + resFile + " -config " + c.ConfPath + " -monitorPort " + strconv.Itoa(monitorPort) + " &> " + logFile + "_" + strconv.Itoa(run)
}

// Configure copies files form the shared directory to slave local storage
func (c SlaveCommands) Configure(masterIP string) map[int]string {
	cmds := make(map[int]string)
	cmds[0] = "mkdir -p " + sharedDir
	cmds[1] = "sudo apt-get -y install nfs-common"
	cmds[2] = "sudo mount -t nfs " + masterIP + ":" + sharedDir + " " + sharedDir
	cmds[3] = "cp -r " + sharedDir + "/* " + "/tmp"
	return cmds
}

//CopyRegistryFileFromSharedDirToLocalStorage
func (c SlaveCommands) CopyRegistryFileFromSharedDirToLocalStorage() map[int]string {
	cmds := make(map[int]string)
	cmds[0] = "cp " + sharedDir + "/aws.csv" + " /tmp"
	cmds[1] = "chmod 777 " + c.SlaveBinPath
	return cmds
}

// Start starts executable
func (c SlaveCommands) start(masterAddr, sync string, monitorAddr, ids string, run int) string {
	return c.SlaveBinPath + " -config " + c.ConfPath + " -registry " + c.RegPath + " -monitor " + monitorAddr + " -master " + masterAddr + ids + " -sync " + sync + " -run " + strconv.Itoa(run)
}

func (c SlaveCommands) Start(masterAddr, monitorAddr string, inst Instance, run int) string {
	startBuilder := newCmdbuilder(c.SameBinary, c.SyncBasePort)

	idsAndSyncLS := startBuilder.startSlave(inst)
	var strCmds []string
	for _, l := range idsAndSyncLS {
		ids := strings.Join(l.ids, " ")
		start := c.start(masterAddr, l.sync, monitorAddr, ids, run)
		strCmds = append(strCmds, start)
	}
	return strings.Join(strCmds, " & ")
}
