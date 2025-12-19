/*
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
* Project: Dingoadm
* Author: jackblack369 (Dongwei)
 */

package monitor

import (
	"fmt"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/task/scripts"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/internal/task/task/common"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
)

func NewSyncHostsMappingTask(dingoadm *cli.DingoAdm, cfg *configure.MonitorConfig) (*task.Task, error) {
	role := cfg.GetRole()
	if role != ROLE_PROMETHEUS {
		return nil, nil
	}
	serviceId := dingoadm.GetServiceId(cfg.GetId())
	containerId, err := dingoadm.GetContainerId(serviceId)
	if err != nil {
		return nil, err
	}

	host := cfg.GetHost()
	hc, err := dingoadm.GetHost(host)
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s role=%s containerId=%s",
		host, role, tui.TrimContainerId(containerId))
	t := task.NewTask("Sync Host Mapping", subname, hc.GetSSHConfig())
	// add step to task
	var out string
	t.AddStep(&step.ListContainers{ // gurantee container exist
		ShowAll:     true,
		Format:      `"{{.ID}}"`,
		Filter:      fmt.Sprintf("id=%s", containerId),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: common.CheckContainerExist(cfg.GetHost(), cfg.GetRole(), containerId, &out),
	})

	// extract host /etc/hosts ip mapping into prometheus container
	mutate_hosts := fmt.Sprintf(cfg.GetConfDir() + "/mutate_hosts")

	t.AddStep(&step.InstallFile{
		HostDestPath: fmt.Sprintf("%s/extract_hosts.sh", cfg.GetConfDir()),
		Content:      &scripts.EXTRACT_HOSTS,
		ExecOptions:  dingoadm.ExecOptions(),
	})
	// append extracted hosts into container /etc/hosts
	t.AddStep(&step.Command{
		Command:     fmt.Sprintf("bash %s/extract_hosts.sh %s", cfg.GetConfDir(), mutate_hosts),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	// docker exec to append hosts into container /etc/hosts
	t.AddStep(&step.ContainerExec{
		ContainerId: &containerId,
		Command:     fmt.Sprintf("sh -c 'cat /etc/prometheus/%s >> /etc/hosts'", "mutate_hosts"),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})

	return t, nil
}
