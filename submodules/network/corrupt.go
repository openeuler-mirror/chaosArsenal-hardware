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
	var newFaultType = corrupt{
		FaultType: "network-corrupt",
	}
	submodules.Add(newFaultType.FaultType, &newFaultType)
}

type corrupt struct {
	FaultType string
	base      baseInfo
}

func (c *corrupt) Prepare(inputArgs []string) error {
	return c.base.Init(inputArgs)
}

func (c *corrupt) FaultInject(_ []string) error {
	return c.base.Executor()
}

func (c *corrupt) FaultRemove(_ []string) error {
	return c.base.Executor()
}
