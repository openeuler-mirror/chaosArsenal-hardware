/*
Copyright 2023 Sangfor Technologies Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package network

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"arsenal-hardware/internal/parse"
	"arsenal-hardware/submodules"
	"arsenal-hardware/util"
)

type iptablesCtl struct {
	inputFlags   map[string]string
	nicDevice    string
	protocol     string
	opsType      string
	chain        string
	interfaceKey string
	shellCmd     string
}

func (i *iptablesCtl) iptablesCtlParamsInit(inputArgs []string) error {
	i.opsType = inputArgs[submodules.OpsTypeIndex]

	flags := parse.TransInputFlagsToMap(inputArgs)
	if nicDevice, ok := flags["interface"]; ok {
		i.nicDevice = nicDevice
	}
	if chain, ok := flags["chain"]; ok {
		i.chain = chain
	}
	if protocol, ok := flags["protocol"]; ok {
		i.protocol = protocol
	}
	i.inputFlags = flags
	return nil
}

func (i *iptablesCtl) interfaceIsExist() bool {
	if i.nicDevice == "" {
		return false
	}
	_, err := net.InterfaceByName(i.nicDevice)
	return err == nil
}

// dependentsCmdCheck 检查环境是否存在iptables命令。
func (i *iptablesCtl) dependentsCmdCheck() error {
	if missingCmd, isMissCmd := util.CheckEnvShellCommand([]string{"iptables"}); isMissCmd {
		return fmt.Errorf("missing command: %s", missingCmd)
	}
	return nil
}

func (i *iptablesCtl) removeArgValue(input string, arg string) string {
	// 使用正则表达式匹配参数及其后的值，并替换为空字符串。
	re := regexp.MustCompile(fmt.Sprintf(`%s\s+\S+`, arg))
	return re.ReplaceAllString(input, "")
}

func (i *iptablesCtl) getInterfaceKey() {
	// TODO: chain类型为FORWARD既支持--in-interface也支持--out-interface，
	// 详细信息参考iptables man手册，本项目默认设定为--in-interface。
	switch i.chain {
	case "INPUT", "PREROUTING":
		i.interfaceKey = "--in-interface"
	case "OUTPUT", "POSTROUTING":
		i.interfaceKey = "--out-interface"
	default:
		i.interfaceKey = "--in-interface"
	}
}

func (i *iptablesCtl) setShellCmd(inputArgs []string) error {
	// iptables -A INPUT --protocol icmp -j DROP -i eth0 --source $sip --source-port
	// $sport --destination $dip --destination-port $dport
	if i.chain == "" {
		return fmt.Errorf("missing param: chain")
	}
	flagsString := parse.TransInputFlagsToString(inputArgs)

	// --chain 需要替换成-A $chain；
	// --protocol 需要放在所有参数的前面；
	// --interface 需要根据具体的chain类型选定相应的参数；
	newInput := flagsString
	argsToRemove := []string{"--chain", "--protocol", "--interface"}
	for _, arg := range argsToRemove {
		newInput = i.removeArgValue(newInput, arg)
	}
	i.getInterfaceKey()

	var shellCmd string
	shellCmd = fmt.Sprintf("iptables -A %s --protocol %s %s %s %s -j DROP", i.chain,
		i.protocol, i.interfaceKey, i.nicDevice, newInput)
	if i.opsType == submodules.Remove {
		shellCmd = strings.Replace(shellCmd, "-A", "-D", 1)
	}
	i.shellCmd = shellCmd
	return nil
}
