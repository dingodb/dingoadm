/*
*  Copyright (c) 2023 NetEase Inc.
*  Copyright (c) 2025 dingodb.com.
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
 */

/*
* Project: Curveadm
* Created Date: 2023-04-28
* Author: wanghai (SeanHai)
*
* Project: Dingoadm
* Author: jackblack369 (Dongwei)
 */

package monitor

import (
	"fmt"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/internal/task/task/common"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
)

func NewRestartServiceTask(dingoadm *cli.DingoAdm, cfg *configure.MonitorConfig) (*task.Task, error) {
	serviceId := dingoadm.GetServiceId(cfg.GetId())
	containerId, err := dingoadm.GetContainerId(serviceId)
	if IsSkip(cfg, []string{ROLE_MONITOR_CONF}) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	hc, err := dingoadm.GetHost(cfg.GetHost())
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s role=%s containerId=%s",
		cfg.GetHost(), cfg.GetRole(), tui.TrimContainerId(containerId))
	t := task.NewTask("Restart Monitor Service", subname, hc.GetSSHConfig())

	// add step to task
	var out string
	var success bool
	host, role := cfg.GetHost(), cfg.GetRole()
	t.AddStep(&step.ListContainers{
		ShowAll:     true,
		Format:      `"{{.ID}}"`,
		Filter:      fmt.Sprintf("id=%s", containerId),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: common.CheckContainerExist(host, role, containerId, &out),
	})
	t.AddStep(&step.RestartContainer{
		ContainerId: containerId,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: common.WaitContainerStart(3),
	})
	t.AddStep(&common.Step2CheckPostStart{
		Host:        host,
		ContainerId: containerId,
		Success:     &success,
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})

	return t, nil
}
