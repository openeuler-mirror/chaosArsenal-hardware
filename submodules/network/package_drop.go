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

	"arsenal-hardware/submodules"
	"arsenal-hardware/util"
)

func init() {
	var newFaultType = packageDrop{
		FaultType: "network-package-drop",
	}
	submodules.Add(newFaultType.FaultType, &newFaultType)
}

type packageDrop struct {
	FaultType   string
	iptablesCtl iptablesCtl
}

func (p *packageDrop) Prepare(inputArgs []string) error {
	if err := p.iptablesCtl.dependentsCmdCheck(); err != nil {
		return errors.New("not found iptables command in current environment")
	}

	if err := p.iptablesCtl.iptablesCtlParamsInit(inputArgs); err != nil {
		return err
	}
	if !p.iptablesCtl.interfaceIsExist() {
		return fmt.Errorf("not found interface %s", p.iptablesCtl.nicDevice)
	}
	return nil
}

func (p *packageDrop) runShellCmd() error {
	if result, err := util.ExecCommandBlock(p.iptablesCtl.shellCmd); err != nil {
		return fmt.Errorf("run cmd(%s) failed(%v), result(%s)", p.iptablesCtl.shellCmd, err, result)
	}
	return nil
}

func (p *packageDrop) FaultInject(inputArgs []string) error {
	if err := p.iptablesCtl.setShellCmd(inputArgs); err != nil {
		return fmt.Errorf("%s set shell cmd failed(%v)", p.FaultType, err)
	}
	return p.runShellCmd()
}

func (p *packageDrop) FaultRemove(inputArgs []string) error {
	if err := p.iptablesCtl.setShellCmd(inputArgs); err != nil {
		return fmt.Errorf("%s set shell cmd failed(%v)", p.FaultType, err)
	}
	return p.runShellCmd()
}
