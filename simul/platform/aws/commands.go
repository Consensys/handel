package aws

import (
	"strconv"
	"strings"
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
func (c MasterCommands) Start(masterAddr string, nbOfNodes, nbOffline, nbOfInstances, timeOut int, run int, threshold int, network string, resFile string, monitorPort int) string {
	return "nohup " + c.MasterBinPath + " -masterAddr " + masterAddr + " -nbOfNodes " + strconv.Itoa(nbOfNodes) + " -nbOffline " + strconv.Itoa(nbOffline) + " -nbOfInstances " + strconv.Itoa(nbOfInstances) + " -timeOut " + strconv.Itoa(timeOut) + " -run " + strconv.Itoa(run) + " -threshold " + strconv.Itoa(threshold) + " -network " + network + " -resultFile " + resFile + " -monitorPort " + strconv.Itoa(monitorPort) + " &> " + logFile + "_" + strconv.Itoa(run)
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
func (c SlaveCommands) Start(masterAddr, sync string, monitorAddr, ids string, run int, log string) string {
	return "nohup " + c.SlaveBinPath + " -config " + c.ConfPath + " -registry " + c.RegPath + " -monitor " + monitorAddr + " -master " + masterAddr + ids + " -sync " + sync + " -run " + strconv.Itoa(run) + " &> " + log + " &"
}

func cmdMapToString(cmds map[int]string) string {
	c := make([]string, 0, len(cmds))
	for idx := 0; idx < len(cmds); idx++ {
		c = append(c, cmds[idx])
	}
	return strings.Join(c, " && ")
}
