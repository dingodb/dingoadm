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
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

// __SIGN_BY_WINE93__

package common

import (
	"fmt"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
)

func CheckContainerExist(host, role, containerId string, out *string) step.LambdaType {
	return func(ctx *context.Context) error {
		if len(*out) == 0 {
			//return errno.ERR_CONTAINER_ALREADT_REMOVED.
			//	F("host=%s role=%s containerId=%s",
			//		host, role, tui.TrimContainerId(containerId))
			return task.ERR_SKIP_TASK
		}
		return nil
	}
}

func checkContainerId(containerId string) step.LambdaType {
	return func(ctx *context.Context) error {
		if containerId == comm.CLEANED_CONTAINER_ID { // container has removed
			return task.ERR_SKIP_TASK
		}
		return nil
	}
}

func NewStopServiceTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	hc, err := dingoadm.GetHost(dc.GetHost())
	if err != nil {
		return nil, err
	}
	host, role := dc.GetHost(), dc.GetRole()
	if role == topology.ROLE_FS_MDS_CLI {
		skipTmp := dingoadm.MemStorage().Get(comm.KEY_SKIP_MDSV2_CLI)
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

	// new task
	subname := fmt.Sprintf("host=%s role=%s containerId=%s",
		dc.GetHost(), dc.GetRole(), tui.TrimContainerId(containerId))
	t := task.NewTask("Stop Service", subname, hc.GetSSHConfig())

	// add step to task
	var out string
	t.AddStep(&step.Lambda{
		Lambda: checkContainerId(containerId),
	})
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
	t.AddStep(&step.StopContainer{
		ContainerId: containerId,
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	return t, nil
}
