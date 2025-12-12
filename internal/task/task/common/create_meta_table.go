/*
 * 	Copyright (c) 2025 dingodb.com Inc.
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
package common

import (
	"fmt"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
)

// NewCreateMetaTablesTask create meta tables in dingo-store
func NewCreateMetaTablesTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
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
	t := task.NewTask("Create Meta Tables", "Create Meta Tables", hc.GetSSHConfig())

	// add step to task
	var success bool
	var out string
	t.AddStep(&step.ListContainers{
		ShowAll:     true,
		Format:      `"{{.ID}}"`,
		Filter:      fmt.Sprintf("id=%s", containerId),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: CheckContainerExist(dc.GetHost(), dc.GetRole(), containerId, &out),
	})

	t.AddStep(&step.ContainerExec{
		Command: fmt.Sprintf("bash %s/%s %s", dc.GetProjectLayout().FSMdsCliBinDir, topology.SCRIPT_CREATE_MDSV2_TABLES, dc.GetProjectLayout().FSMdsCliBinaryPath),
		//Command:     fmt.Sprintf("bash %s/create_mdsv2_tables.sh %s/dingo-mds-client", dc.GetProjectLayout().DingoStoreBinDir, dc.GetProjectLayout().DingoStoreBinDir),
		ContainerId: &containerId,
		Success:     &success,
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})

	t.AddStep(&step.Lambda{
		Lambda: checkCreateTableSuccess(&success, &out),
	})

	return t, nil
}

func checkCreateTableSuccess(success *bool, out *string) step.LambdaType {
	return func(ctx *context.Context) error {
		if !*success {
			return errno.ERR_CREATE_META_TABLE_FAILED.S(*out)
		}
		return nil
	}
}
