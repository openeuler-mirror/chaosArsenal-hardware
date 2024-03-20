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
	"arsenal-hardware/submodules"
)

func init() {
	var newFaultType = duplicate{
		FaultType: "network-duplicate",
	}
	submodules.Add(newFaultType.FaultType, &newFaultType)
}

type duplicate struct {
	FaultType string
	base      baseInfo
}

func (d *duplicate) Prepare(inputArgs []string) error {
	return d.base.Init(inputArgs)
}

func (d *duplicate) FaultInject(_ []string) error {
	return d.base.Executor()
}

func (d *duplicate) FaultRemove(_ []string) error {
	return d.base.Executor()
}
