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
	var newFaultType = resetAbnormal{
		FaultType: "pcie-reset-abnormal",
	}
	submodules.Add(newFaultType.FaultType, &newFaultType)
}

type resetAbnormal struct {
	FaultType string
	pcie      pcie
}

func (r *resetAbnormal) Prepare(inputArgs []string) error {
	return r.pcie.preCheck(inputArgs)
}

func (r *resetAbnormal) FaultInject(_ []string) error {
	// 判断pcie设备是否存在。
	if !r.pcie.pcieDeviceIsExist() {
		return fmt.Errorf("not fond pcie device: %s", r.pcie.bdf)
	}

	// 判断pcie设备是否支持reset操作。
	if !r.pcie.pcieDeviceIsSupportReset() {
		return fmt.Errorf("device(%s) not support pcie reset", r.pcie.bdf)
	}

	// reset目标pcie设备。
	return r.pcie.triggerPcieRefOps("reset", r.pcie.bdf)
}

func (r *resetAbnormal) FaultRemove(_ []string) error {
	return nil
}
