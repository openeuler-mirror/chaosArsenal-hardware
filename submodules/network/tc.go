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
	"strings"
	"time"

	"arsenal-hardware/internal/parse"
	"arsenal-hardware/submodules"
	"arsenal-hardware/util"
)

var (
	filterFlags = []string{"source", "source-port", "destination", "destination-port"}
	tcOps       = map[string]string{
		submodules.Inject: "add",
		submodules.Remove: "del",
	}
)

type baseInfo struct {
	flags     map[string]string
	opsType   string
	faultType string
	tcOpsType string
}

func (b *baseInfo) Init(inputArgs []string) error {
	dependCmd := []string{"tc", "lsmod", "grep", "modprobe"}
	if missingCmd, isMissCmd := util.CheckEnvShellCommand(dependCmd); isMissCmd {
		return fmt.Errorf("missing command: %s", missingCmd)
	}

	checkModuleCmd := "lsmod | grep sch_netem"
	retString, err := util.ExecCommandBlock(checkModuleCmd)
	// 如果没有抓到对应关键字，返回值为1。
	if err != nil && retString == " " {
		return fmt.Errorf("execute: %s failed: %s, result: %s", checkModuleCmd, err, retString)
	}

	if !strings.Contains(retString, "sch_netem") {
		loadModuleCmd := "modprobe sch_netem"
		if result, err := util.ExecCommandBlock(loadModuleCmd); err != nil {
			return fmt.Errorf("execute shell command: %s failed, error: %s, result: %s",
				loadModuleCmd, err, result)
		}
	}

	b.flags = parse.TransInputFlagsToMap(inputArgs)
	b.opsType = inputArgs[submodules.OpsTypeIndex]
	b.faultType = inputArgs[submodules.FaultTypeIndex]
	return nil
}

// Executor 执行tc命令。
func (b *baseInfo) Executor() error {
	shellCommands, err := b.getTcExecuteShellCmd()
	if err != nil {
		return fmt.Errorf("get tc fault inject shell command failed: %v", err)
	}

	const interval = 100
	for i := 0; i < len(shellCommands); i++ {
		if result, err := util.ExecCommandBlock(shellCommands[i]); err != nil {
			return fmt.Errorf("execute: %s failed: %s, result: %s", shellCommands[i], err, result)
		}
		time.Sleep(interval * time.Millisecond)
	}
	return nil
}

// shouldAddTcFilter 判断是否需要添加tc filter。
func (b *baseInfo) shouldAddTcFilter() bool {
	for i := 0; i < len(filterFlags); i++ {
		if b.flags[filterFlags[i]] != "" {
			return true
		}
	}
	return false
}

func (b *baseInfo) iterFilterCmd() string {
	var cmd string
	if nicDevice, ok := b.flags["interface"]; ok {
		cmd = fmt.Sprintf("tc filter add dev %s protocol ip parent 1:0 prio 4 u32", nicDevice)
	}

	var matchArg string
	for i := 0; i < len(filterFlags); i++ {
		value, ok := b.flags[filterFlags[i]]
		if !ok {
			continue
		}
		switch filterFlags[i] {
		case "source":
			if sourceSubnetMask, ok := b.flags["source-subnet-mask"]; ok {
				matchArg = fmt.Sprintf("match ip src %s/%s", value, sourceSubnetMask)
			} else {
				matchArg = fmt.Sprintf("match ip src %s", value)
			}
		case "destination":
			if destinationSubnetMask, ok := b.flags["destination-subnet-mask"]; ok {
				matchArg = fmt.Sprintf("match ip dst %s/%s", value, destinationSubnetMask)
			} else {
				matchArg = fmt.Sprintf("match ip dst %s", value)
			}
		case "source-port":
			matchArg = fmt.Sprintf("match ip sport %s 0xffff", value)
		case "destination-port":
			matchArg = fmt.Sprintf("match ip dport %s 0xffff", value)
		default:
			break
		}
		if matchArg != "" {
			cmd = fmt.Sprintf("%s %s", cmd, matchArg)
			matchArg = ""
		}
	}
	return fmt.Sprintf("%s flowid 1:4", cmd)
}

// getTcFilterCommands 如果需要设定tc过滤器，需要返回三条shell命令。
func (b *baseInfo) getTcFilterCommands() []string {
	var shellCommands []string
	switch b.faultType {
	case "delay":
		// tc qdisc add dev ens18 root handle 1: prio bands 4
		nicDevice, ok := b.flags["interface"]
		if !ok {
			break
		}
		tcOpsType := b.tcOpsType
		cmd1 := fmt.Sprintf("tc qdisc %s dev %s root handle 1: prio bands 4",
			tcOpsType, nicDevice)
		shellCommands = append(shellCommands, cmd1)

		// 如果是清理命令，直接将root qdisc移除即可。
		if b.opsType == submodules.Remove {
			break
		}

		delayTime, ok := b.flags["delay"]
		if !ok {
			break
		}
		// tc qdisc add dev ens18 parent 1:4 handle 40: netem delay 100ms
		cmd2 := fmt.Sprintf("tc qdisc %s dev %s parent 1:4 handle 40: netem delay %s",
			tcOpsType, nicDevice, delayTime)
		shellCommands = append(shellCommands, cmd2)

		// tc filter add dev ens18 protocol ip parent 1:0 prio 4 u32
		// match ip dst 10.103.176.207 match ip dport 22 0xffff flowid 1:4
		cmd3 := b.iterFilterCmd()
		shellCommands = append(shellCommands, cmd3)
	default:
		break
	}
	return shellCommands
}

func (b *baseInfo) getTcExecuteShellCmd() ([]string, error) {
	var shellCommands []string
	tcOperation, ok := tcOps[b.opsType]
	if !ok {
		return nil, fmt.Errorf("not support fault operation: %s", b.opsType)
	}
	b.tcOpsType = tcOperation

	nicDevice, ok := b.flags["interface"]
	if !ok {
		return nil, fmt.Errorf("missing param: interface")
	}
	switch b.faultType {
	case "loss", "corrupt", "duplicate":
		percentStr, ok := b.flags["percent"]
		if !ok {
			return nil, fmt.Errorf("missing param: percent")
		}
		cmd := fmt.Sprintf("tc qdisc %s dev %s root netem %s %s",
			b.tcOpsType, nicDevice, b.faultType, percentStr)
		shellCommands = append(shellCommands, cmd)
	case "delay":
		if b.shouldAddTcFilter() {
			shellCommands = b.getTcFilterCommands()
		} else {
			timeStr, ok := b.flags["delay"]
			if !ok {
				return nil, fmt.Errorf("missing param: delay")
			}
			cmd := fmt.Sprintf("tc qdisc %s dev %s root netem delay %s", b.tcOpsType, nicDevice, timeStr)
			shellCommands = append(shellCommands, cmd)
		}
	case "reorder":
		timeStr, ok := b.flags["delay"]
		if !ok {
			return nil, fmt.Errorf("missing param: delay")
		}
		percentStr, ok := b.flags["percent"]
		if !ok {
			return nil, fmt.Errorf("missing param: percent")
		}
		relatperStr, ok := b.flags["relatper"]
		if !ok {
			return nil, fmt.Errorf("missing param: relatper")
		}
		cmd := fmt.Sprintf("tc qdisc %s dev %s root netem delay %s %s %s %s",
			b.tcOpsType, nicDevice, timeStr, b.faultType, percentStr, relatperStr)
		shellCommands = append(shellCommands, cmd)
	default:
		return shellCommands, fmt.Errorf("unsupported fault type: %s", b.faultType)
	}
	return shellCommands, nil
}
