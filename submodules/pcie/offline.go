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

package pcie

import (
	"fmt"

	"arsenal-hardware/submodules"
)

func init() {
	var newFaultType = offline{
		FaultType: "pcie-offline",
	}
	submodules.Add(newFaultType.FaultType, &newFaultType)
}

type offline struct {
	FaultType string
	pcie      pcie
}

func (o *offline) Prepare(inputArgs []string) error {
	return o.pcie.preCheck(inputArgs)
}

func (o *offline) FaultInject(_ []string) error {
	// 查找输入pcie设备的pcie root bus。
	if err := o.pcie.findPcieDeviceRootBus(); err != nil {
		return fmt.Errorf("find pcie device root bus failed(%v)", err)
	}

	// 备份pcie设备与pcie root bus信息。
	if err := o.pcie.backupPcieRootBusInfo(); err != nil {
		return fmt.Errorf("backup root bus info failed(%v)", err)
	}

	// 移除目标pcie设备。
	if err := o.pcie.triggerPcieRefOps("remove", o.pcie.bdf); err != nil {
		if err := o.pcie.removePcieRootBusInfo(); err != nil {
			return fmt.Errorf("remove pcie backup root bus info file failed(%v)", err)
		}
		return fmt.Errorf("trigger pcie device offline failed(%v)", err)
	}
	return nil
}

func (o *offline) FaultRemove(_ []string) error {
	// 根据输入pcie的bdf信息扫描arsenal/logs/目录，获取pcie root bus。
	if err := o.pcie.getBackupPcieRootBusViaFilePath(); err != nil {
		return fmt.Errorf("get pcie root bus via backup file path failed(%v)", err)
	}

	if err := o.pcie.triggerPcieRefOps("rescan", o.pcie.rootBus); err != nil {
		return fmt.Errorf("scan root bus failed(%v)", err)
	}
	return o.pcie.removePcieRootBusInfo()
}
