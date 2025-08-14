/*
 *  Copyright (c) 2021 NetEase Inc.
 * 	Copyright (c) 2024 dingodb.com Inc.
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
 * Project: CurveAdm
 * Created Date: 2021-10-15
 * Author: Jingli Chen (Wine93)
 */

// __SIGN_BY_WINE93__

package common

import (
	"fmt"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/pkg/module"
)

const (
	CMD_ADD_CONTABLE = "bash -c '[[ ! -z $(which crontab) ]] && crontab %s'"
)

type Step2CheckPostStart struct {
	Host        string
	Role        string
	ContainerId string
	Success     *bool
	Out         *string
	ExecOptions module.ExecOptions
}

func (s *Step2CheckPostStart) Execute(ctx *context.Context) error {
	if *s.Success {
		return nil
	}

	var status string
	step := &step.InspectContainer{
		ContainerId: s.ContainerId,
		Format:      "'{{.State.Status}}'",
		Out:         &status,
		ExecOptions: s.ExecOptions,
	}
	err := step.Execute(ctx)
	if err != nil {
		return errno.ERR_START_CRONTAB_IN_CONTAINER_FAILED.S(*s.Out)
	} else if status != "running" {
		return errno.ERR_CONTAINER_IS_ABNORMAL.
			F("host=%s role=%s containerId=%s",
				s.Host, s.Role, tui.TrimContainerId(s.ContainerId))
	}
	return nil
}

func NewStartServiceTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	if dc.GetRole() == topology.ROLE_MDSV2_CLI {
		skipTmp := dingoadm.MemStorage().Get(comm.KEY_SKIP_TMP)
		if skipTmp != nil && skipTmp.(bool) {
			return nil, nil
		}
	}

	serviceId := dingoadm.GetServiceId(dc.GetId())
	containerId, err := dingoadm.GetContainerId(serviceId)
	if dingoadm.IsSkip(dc) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	hc, err := dingoadm.GetHost(dc.GetHost())
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s role=%s containerId=%s",
		dc.GetHost(), dc.GetRole(), tui.TrimContainerId(containerId))
	t := task.NewTask("Start Service", subname, hc.GetSSHConfig())

	// add step to task
	var out string
	var success bool
	host, role := dc.GetHost(), dc.GetRole()
	t.AddStep(&step.ListContainers{
		ShowAll:     true,
		Format:      `"{{.ID}}"`,
		Filter:      fmt.Sprintf("id=%s", containerId),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: CheckContainerExist(host, role, containerId, &out),
	})
	t.AddStep(&step.StartContainer{
		ContainerId: &containerId,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: WaitContainerStart(3),
	})
	//t.AddStep(&step.ContainerExec{
	//	ContainerId: &containerId,
	//	Command:     fmt.Sprintf(CMD_ADD_CONTABLE, CURVE_CRONTAB_FILE),
	//	Success:     &success,
	//	Out:         &out,
	//	ExecOptions: curveadm.ExecOptions(),
	//})
	t.AddStep(&Step2CheckPostStart{
		Host:        dc.GetHost(),
		Role:        dc.GetRole(),
		ContainerId: containerId,
		Success:     &success,
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})

	return t, nil
}
