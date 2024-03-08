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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"arsenal-hardware/internal/parse"
	"arsenal-hardware/util"
)

type pcie struct {
	bdf                          string
	rootBus                      string
	bdfInfoBackupDir             string
	backupBdfRootBusInfoFilePath string
}

func (p *pcie) dependentsCmdCheck() error {
	if missingCmd, isMissCmd := util.CheckEnvShellCommand([]string{"echo"}); isMissCmd {
		return fmt.Errorf("missing command: %s", missingCmd)
	}
	return nil
}

func (p *pcie) pcieBdfFormatCheck(flags map[string]string) error {
	bdf, ok := flags["bdf"]
	if !ok {
		return errors.New("missing param: bdf")
	}

	// 正则匹配，要求输入带domain number的pcie bdf信息。
	re := regexp.MustCompile(`^[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-9a-f]{1}$`)
	if !re.MatchString(bdf) {
		return errors.New("invalid bpf format, example: 0000:00:02.0")
	}
	p.bdf = bdf
	return nil
}

func (p *pcie) preCheck(inputArgs []string) error {
	if err := p.dependentsCmdCheck(); err != nil {
		return err
	}
	return p.pcieBdfFormatCheck(parse.TransInputFlagsToMap(inputArgs))
}

func (p *pcie) pcieDeviceIsExist() bool {
	attrPath := fmt.Sprintf("/sys/bus/pci/devices/%s", p.bdf)
	return util.FileIsExist(attrPath)
}

func (p *pcie) pcieDeviceIsSupportReset() bool {
	attrPath := fmt.Sprintf("/sys/bus/pci/devices/%s/reset", p.bdf)
	return util.FileIsExist(attrPath)
}

func (p *pcie) triggerPcieRefOps(opsType string, bdf string) error {
	var controlPath string
	switch opsType {
	case "reset", "remove":
		// /sys/bus/pci/devices下列举了所有pcie设备，不必关心root bus。
		controlPath = fmt.Sprintf("/sys/bus/pci/devices/%s/%s", bdf, opsType)
	case "rescan":
		controlPath = fmt.Sprintf("/sys/devices/pci%s/pci_bus/%s/%s", bdf, bdf, opsType)
	default:
		return errors.New("please input valid pcie trigger ops type")
	}

	if !util.FileIsExist(controlPath) {
		return fmt.Errorf("bfd: %s trigger control path: %s not found", bdf, controlPath)
	}

	shellCmd := fmt.Sprintf("echo 1 > %s", controlPath)
	result, err := util.ExecCommandBlock(shellCmd)
	if err != nil {
		return fmt.Errorf("execute: %s failed: %v result: %s", shellCmd, err, result)
	}
	return nil
}

func (p *pcie) findPcieDeviceRootBus() error {
	DirPath := fmt.Sprintf("/sys/bus/pci/devices/%s", p.bdf)
	if !util.FileIsExist(DirPath) {
		return fmt.Errorf("not found device attr dir(%s)", DirPath)
	}

	link, err := os.Readlink(DirPath)
	if err != nil {
		return fmt.Errorf("get device attr dir(%s) link failed", DirPath)
	}

	re := regexp.MustCompile(`[0-9a-fA-F]{4}:[0-9a-fA-F]{2}`)
	pcieRootBus := re.FindString(link)
	if pcieRootBus == "" {
		return fmt.Errorf("get root bus from link(%s) failed", link)
	}
	p.rootBus = pcieRootBus
	return nil
}

func (p *pcie) splicePcieRootBusBackupPath() (string, error) {
	// 信息备份在arsenal/logs/pcie/$root_bus-$bdf文件。
	arsenalLogDir, err := util.GetArsenalLogsDir()
	if err != nil {
		return "", fmt.Errorf("get arsenal logs dir failed(%v)", err)
	}
	if !util.FileIsExist(arsenalLogDir) {
		return "", fmt.Errorf("not found arsenal logs dir(%s)", arsenalLogDir)
	}
	p.bdfInfoBackupDir = arsenalLogDir
	return fmt.Sprintf("%s/pcie-%s-%s", arsenalLogDir, p.rootBus, p.bdf), nil
}

func (p *pcie) backupPcieRootBusInfo() error {
	// 判断pcie设备是否存在。
	if !p.pcieDeviceIsExist() {
		return fmt.Errorf("pcie device(%s) not exist", p.bdf)
	}

	// 判断pcie设备root bus备份文件是否存在。
	backupInfoFilePath, err := p.splicePcieRootBusBackupPath()
	if err != nil {
		return fmt.Errorf("splice root bus backup info file path failed(%v)", err)
	}
	if util.FileIsExist(backupInfoFilePath) {
		return fmt.Errorf("backup root bus info file has exist, double inject")
	}

	// 遍历arsenal/logs/文件夹下是否存在与当前目标pcie设备同root bus的pcie设备。
	var matchFilePath string
	preFix := fmt.Sprintf("pcie-%s", p.rootBus)
	err = filepath.Walk(p.bdfInfoBackupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasPrefix(info.Name(), preFix) {
			matchFilePath = path
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("traversing the folder: %v", err)
	}
	// 如果存在则不允许被注入故障，因为恢复的时候是通过扫描pcie设备root bus，
	// 同一个root bus下的设备都会被恢复，影响恢复逻辑。
	if matchFilePath != "" {
		return fmt.Errorf("faults have been injected under the pcie root bus: %s", p.rootBus)
	}

	// 创建pcie设备与root bus信息的备份文件。
	file, err := os.Create(backupInfoFilePath)
	if err != nil {
		return fmt.Errorf("create root bus backup file failed(%v)", err)
	}
	defer file.Close()
	return nil
}

func (p *pcie) getRootBusViaFilePath() string {
	// 信息备份在arsenal/logs/pcie/$root_bus-$bdf文件，备份文件名示例：pcie-0000:00-0000:00:18.7。
	var lengthAfterSplit = 2
	parts := strings.Split(filepath.Base(p.backupBdfRootBusInfoFilePath), "-")
	if len(parts) >= lengthAfterSplit {
		return parts[1]
	}
	return ""
}

func (p *pcie) getBackupPcieRootBusViaFilePath() error {
	arsenalLogDir, err := util.GetArsenalLogsDir()
	if err != nil {
		return fmt.Errorf("get arsenal logs dir failed(%v)", err)
	}
	if !util.FileIsExist(arsenalLogDir) {
		return fmt.Errorf("not found arsenal logs dir(%s)", arsenalLogDir)
	}
	p.bdfInfoBackupDir = arsenalLogDir

	// 遍历arsenal/logs/文件夹下是否存在已经注入故障bdf信息。
	var matchFilePath string
	sufFix := fmt.Sprintf("-%s", p.bdf)
	err = filepath.Walk(p.bdfInfoBackupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), sufFix) {
			matchFilePath = path
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("query dir(%s) error(%v)", arsenalLogDir, err)
	}
	if matchFilePath == "" {
		return fmt.Errorf("please check device(%s) has inject fault", p.bdf)
	}
	p.backupBdfRootBusInfoFilePath = matchFilePath

	parseRootBus := p.getRootBusViaFilePath()
	if parseRootBus == "" {
		return fmt.Errorf("get backup root bus failed")
	}
	p.rootBus = parseRootBus
	return nil
}

func (p *pcie) removePcieRootBusInfo() error {
	if p.backupBdfRootBusInfoFilePath == "" {
		return fmt.Errorf("please get backup pcie's root bus info file path first")
	}

	if util.FileIsExist(p.backupBdfRootBusInfoFilePath) {
		if err := os.Remove(p.backupBdfRootBusInfoFilePath); err != nil {
			return fmt.Errorf("remove pcie root bus info backup file failed(%v)", err)
		}
	}
	return nil
}
