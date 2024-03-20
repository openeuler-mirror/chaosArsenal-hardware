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

	"arsenal-hardware/internal/parse"
	"arsenal-hardware/submodules"
	"arsenal-hardware/util"
)

func init() {
	var newFaultType = down{
		FaultType: "network-down",
	}
	submodules.Add(newFaultType.FaultType, &newFaultType)
}

type down struct {
	FaultType string
	flags     map[string]string
	downCmd   string
	upCmd     string
}

func (d *down) setShellCmd() error {
	nicDevice, ok := d.flags["interface"]
	if !ok {
		return fmt.Errorf("%s missing param: interface", d.FaultType)
	}

	// 优先选择nmcli命令构造网卡down。
	if _, isMissCmd := util.CheckEnvShellCommand([]string{"nmcli"}); !isMissCmd {
		d.downCmd = fmt.Sprintf("nmcli connection down %s", nicDevice)
		d.upCmd = fmt.Sprintf("nmcli connection up %s", nicDevice)
		return nil
	}

	if _, isMissCmd := util.CheckEnvShellCommand([]string{"ifconfig"}); !isMissCmd {
		d.downCmd = fmt.Sprintf("ifconfig %s down", nicDevice)
		d.upCmd = fmt.Sprintf("ifconfig %s up", nicDevice)
		return nil
	}

	dependCmd := []string{"ifdown", "ifup"}
	if _, isMissCmd := util.CheckEnvShellCommand(dependCmd); !isMissCmd {
		d.downCmd = fmt.Sprintf("ifdown %s", nicDevice)
		d.upCmd = fmt.Sprintf("ifup %s", nicDevice)
		return nil
	}
	return fmt.Errorf("%s missing command nmcli ifconfig ifdown ifup", d.FaultType)
}

func (d *down) Prepare(inputArgs []string) error {
	d.flags = parse.TransInputFlagsToMap(inputArgs)
	return d.setShellCmd()
}

func (d *down) FaultInject(_ []string) error {
	if result, err := util.ExecCommandBlock(d.downCmd); err != nil {
		return fmt.Errorf("execute: %s failed: %v, result: %s", d.downCmd, err, result)
	}
	return nil
}

func (d *down) FaultRemove(_ []string) error {
	if result, err := util.ExecCommandBlock(d.upCmd); err != nil {
		return fmt.Errorf("execute: %s failed: %v, result: %s", d.upCmd, err, result)
	}
	return nil
}
