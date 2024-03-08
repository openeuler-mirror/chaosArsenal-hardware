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
	"errors"
	"fmt"
	"strings"

	"arsenal-hardware/submodules"
	"arsenal-hardware/util"
)

func init() {
	var newFaultType = unavailable{
		FaultType: "network-unavailable",
	}
	submodules.Add(newFaultType.FaultType, &newFaultType)
}

type unavailable struct {
	FaultType   string
	iptablesCtl iptablesCtl
	cmd         []string
}

func (u *unavailable) setShellCmd() {
	// iptables -A INPUT -i eth0 -j DROP
	// iptables -A OUTPUT -o eth0 -j DROP
	var inputChain, outputChain string
	switch u.iptablesCtl.opsType {
	case submodules.Inject:
		inputChain = fmt.Sprintf("iptables -A INPUT -i %s -j DROP", u.iptablesCtl.nicDevice)
		outputChain = fmt.Sprintf("iptables -A OUTPUT -o %s -j DROP", u.iptablesCtl.nicDevice)
	case submodules.Remove:
		inputChain = fmt.Sprintf("iptables -D INPUT -i %s -j DROP", u.iptablesCtl.nicDevice)
		outputChain = fmt.Sprintf("iptables -D OUTPUT -o %s -j DROP", u.iptablesCtl.nicDevice)
	}
	u.cmd = append(u.cmd, inputChain)
	u.cmd = append(u.cmd, outputChain)
}

func (u *unavailable) restoreEnv(runSuccessCmd []string) error {
	for i := 0; i < len(runSuccessCmd); i++ {
		rawShellCmd := runSuccessCmd[i]
		if rawShellCmd == "" {
			return errors.New("shell cmd is empty")
		}

		var restoreCmd string
		switch u.iptablesCtl.opsType {
		case submodules.Inject:
			restoreCmd = strings.Replace(rawShellCmd, "-A", "-D", 1)
		case submodules.Remove:
			restoreCmd = strings.Replace(rawShellCmd, "-D", "-A", 1)
		}

		// 恢复也报错了，那就没有办法继续往下处理。
		if result, err := util.ExecCommandBlock(restoreCmd); err != nil {
			return fmt.Errorf("%s run restore cmd (%s) failed(%v), result(%s)",
				u.FaultType, restoreCmd, err, result)
		}
	}
	return nil
}

func (u *unavailable) runShellCmd() error {
	// 丢弃chain类型为INPUT&OUTPUT，所以包含两条shell命令。
	var shellCmdLength = 2
	var runSuccessCmd []string
	if len(u.cmd) != shellCmdLength {
		return fmt.Errorf("%s invalid cmd array length", u.FaultType)
	}

	for i := 0; i < len(u.cmd); i++ {
		shellCmd := u.cmd[i]
		if shellCmd == "" {
			return fmt.Errorf("%s shell cmd is empty", u.FaultType)
		}
		if result, err := util.ExecCommandBlock(shellCmd); err != nil {
			if err := u.restoreEnv(runSuccessCmd); err != nil {
				return fmt.Errorf("%s restore env error(%v)", u.FaultType, err)
			}
			return fmt.Errorf("run cmd(%s) failed(%v), result(%s)", shellCmd, err, result)
		}
		runSuccessCmd = append(runSuccessCmd, shellCmd)
	}
	return nil
}

func (u *unavailable) Prepare(inputArgs []string) error {
	if err := u.iptablesCtl.dependentsCmdCheck(); err != nil {
		return fmt.Errorf("not found iptables command in environment")
	}

	if err := u.iptablesCtl.iptablesCtlParamsInit(inputArgs); err != nil {
		return err
	}
	if !u.iptablesCtl.interfaceIsExist() {
		return fmt.Errorf("not found interface %s", u.iptablesCtl.nicDevice)
	}
	return nil
}

func (u *unavailable) FaultInject(_ []string) error {
	u.setShellCmd()
	return u.runShellCmd()
}

func (u *unavailable) FaultRemove(_ []string) error {
	u.setShellCmd()
	return u.runShellCmd()
}
