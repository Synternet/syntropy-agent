package shellcmd

import (
	"bytes"
	"os/exec"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
)

type shellCommandSupportInfo struct {
	name    string
	command string
	params  []string
}

func New(name, cmd string, params ...string) common.SupportInfoHelper {
	scmd := shellCommandSupportInfo{
		name:    name,
		command: cmd,
	}

	scmd.params = append(scmd.params, params...)

	return &scmd
}

func fetchCmdExecOutput(cmdName string, params ...string) string {
	cmd := exec.Command(cmdName, params...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return errb.String()
	}
	return outb.String()
}

func (scmd *shellCommandSupportInfo) SupportInfo() *common.KeyValue {
	return &common.KeyValue{
		Key:   scmd.name,
		Value: fetchCmdExecOutput(scmd.command, scmd.params...),
	}
}
