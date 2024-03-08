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

package all

import (
	// 添加注入清理接口
	_ "arsenal-hardware/internal/operations"
	// 向全局故障相关操作接口map中添加disk类型接口
	_ "arsenal-hardware/submodules/disk"
	// 向全局故障相关操作接口map中添加pcie类型接口
	_ "arsenal-hardware/submodules/pcie"
	// 向全局故障相关操作接口map中添加memory类型接口
	_ "arsenal-hardware/submodules/network"
	// 向全局故障相关操作接口map中添加process类型接口
)
