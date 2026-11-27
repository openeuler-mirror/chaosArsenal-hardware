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

package main

import (
	"fmt"
	"os"

	"arsenal-hardware/pkg/base"
)

// 故障准备：arsenal-hardware prepare disk offline --device sda
// 故障注入：arsenal-hardware inject disk offline --device sda
// 注入清理：arsenal-hardware remove disk offline --device sda
func main() {
	if err := base.Run(os.Args); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
