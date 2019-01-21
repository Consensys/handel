package aws

import (
	"strconv"
	"strings"
)

// Commands represents AWS platform specyfic commands.
type Commands struct {
	MasterBinPath string
	SlaveBinPath  string
	ConfPath      string
	RegPath       string
	S3            string
	copyBinFiles  bool
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
func NewCommands(masterBinPath, slaveBinPath, confPath, regPath, s3 string, copyBinFiles bool) Commands {
	return Commands{
		MasterBinPath: masterBinPath,
		SlaveBinPath:  slaveBinPath,
		ConfPath:      confPath,
		RegPath:       regPath,
		S3:            s3,
		copyBinFiles:  copyBinFiles,
	}
}

func (c MasterCommands) Configure() map[int]string {
	cmds := make(map[int]string)
	cmds[0] = "wget -O " + c.ConfPath + " " + c.S3 + c.ConfPath
	cmds[1] = "chmod 777 " + c.ConfPath
	if c.copyBinFiles {
		cmds[2] = "wget -O " + c.MasterBinPath + " " + c.S3 + c.MasterBinPath
		cmds[3] = "chmod 777 " + c.MasterBinPath
	}
	return cmds
}

//Kill previous run
func (c MasterCommands) Kill() string {
	return "killall " + c.MasterBinPath
}

// Start starts master executable
func (c MasterCommands) Start(masterAddr string, timeOut int, run int, network, resFile string, monitorPort int) string {
	return "nohup " + c.MasterBinPath + " -masterAddr " + masterAddr + " -timeOut " + strconv.Itoa(timeOut) + " -run " + strconv.Itoa(run) + " -network " + network + " -resultFile " + resFile + " -config " + c.ConfPath + " -monitorPort " + strconv.Itoa(monitorPort) + " &> " + logFile + "_" + strconv.Itoa(run)
}

//Kill previous run
func (c SlaveCommands) Kill() string {
	return "killall " + c.SlaveBinPath + " &> kill.log"
}

func (c SlaveCommands) Configure() map[int]string {
	cmds := make(map[int]string)
	cmds[0] = "wget -O " + c.ConfPath + " " + c.S3 + c.ConfPath
	cmds[1] = "chmod 777 " + c.ConfPath
	if c.copyBinFiles {
		cmds[2] = "wget -O " + c.SlaveBinPath + " " + c.S3 + c.SlaveBinPath
		cmds[3] = "chmod 777 " + c.SlaveBinPath
	}
	return cmds
}

func (c SlaveCommands) CopyRegistryFileFromSharedDirToLocalStorage() map[int]string {
	cmds := make(map[int]string)
	cmds[0] = "wget -O " + c.RegPath + " " + c.S3 + c.RegPath
	cmds[1] = "chmod 777 " + c.RegPath
	return cmds
}

func (c SlaveCommands) CopyRegistryFileFromSharedDirToLocalStorageQuitSSH() map[int]string {
	cmds := make(map[int]string)
	cmds[0] = "nohup " + "wget -O " + c.RegPath + " " + c.S3 + c.RegPath + " & " + "chmod 777 " + c.RegPath + " &> cpy.log"
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

func (c SlaveCommands) StartAndQuitSSH(masterAddr, monitorAddr string, inst Instance, run int) string {
	return "nohup " + c.Start(masterAddr, monitorAddr, inst, run) + " &> log.txt"
}
