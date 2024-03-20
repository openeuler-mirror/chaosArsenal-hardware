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

package disk

import (
	"fmt"
	"io/ioutil"

	"arsenal-hardware/util"
)

type disk struct {
	devName      string
	stateCtlPath string
	curState     string
}

func (d *disk) dependentsCmdCheck() error {
	if missingCmd, isMissCmd := util.CheckEnvShellCommand([]string{"echo"}); isMissCmd {
		return fmt.Errorf("missing command: %s", missingCmd)
	}
	return nil
}

func (d *disk) diskStateControlPreRun(flags map[string]string) error {
	if err := d.dependentsCmdCheck(); err != nil {
		return err
	}

	devName, ok := flags["device"]
	if !ok {
		return fmt.Errorf("block device: %s not exist", d.devName)
	}
	d.devName = devName
	if !d.deviceIsExist() {
		return fmt.Errorf("block device: %s not exist", devName)
	}

	if err := d.setStateCtlFilePath(); err != nil {
		return fmt.Errorf("set disk: %s state control file path failed", d.devName)
	}

	if err := d.setCurState(); err != nil {
		return fmt.Errorf("set disk: %s state control file failed(%v)", d.devName, err)
	}
	return nil
}

func (d *disk) deviceIsExist() bool {
	return util.FileIsExist(fmt.Sprintf("/dev/%s", d.devName))
}

func (d *disk) setStateCtlFilePath() error {
	path := fmt.Sprintf("/sys/block/%s/device/state", d.devName)
	if !util.FileIsExist(path) {
		return fmt.Errorf("disk state control file: %s not exist", path)
	}
	d.stateCtlPath = path
	return nil
}

func (d *disk) setCurState() error {
	content, err := ioutil.ReadFile(d.stateCtlPath)
	if err != nil {
		return err
	}

	// 获取到的磁盘状态信息末尾含有换行符。
	d.curState = string(content[:len(content)-1])
	return nil
}

func (d *disk) changeDiskState(state string) error {
	if d.curState == state {
		return fmt.Errorf("disk: %s already in %s state", d.devName, state)
	}

	shellCmd := fmt.Sprintf("echo %s > %s", state, d.stateCtlPath)
	result, err := util.ExecCommandBlock(shellCmd)
	if err != nil {
		return fmt.Errorf("execute: %s failed: %v result: %s", shellCmd, err, result)
	}
	return nil
}
