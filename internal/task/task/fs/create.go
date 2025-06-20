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

package fs

import (
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/internal/utils"
)

func TrimContainerId(containerId *string) step.LambdaType {
	return func(ctx *context.Context) error {
		items := strings.Split(*containerId, "\n")
		*containerId = items[len(items)-1]
		return nil
	}
}

func NewCreateDingoFSTask(dingoadm *cli.DingoAdm, cc *configure.ClientConfig) (*task.Task, error) {
	options := dingoadm.MemStorage().Get(common.KEY_MOUNT_OPTIONS).(MountOptions)
	hc, err := dingoadm.GetHost(options.Host)
	if err != nil {
		return nil, err
	}
	subname := fmt.Sprintf("DingoFS Name=%s, Type=%s", options.MountFSName, options.MountFSType)
	t := task.NewTask("CreateDingoFS", subname, hc.GetSSHConfig())
	containeName := fmt.Sprintf("dingofs-tmp-%s", utils.MD5Sum(options.MountPoint))

	var containerId, out string
	var success bool
	// add create fs step to task
	t.AddStep(&step.CreateContainer{
		Image:      cc.GetContainerImage(),
		Entrypoint: "bash",
		Command:    "-c \"while true; do sleep 3600; done\"",
		Init:       true,
		Name:       containeName,
		Privileged: true,
		Restart:    "no",
		//--ulimit core=-1: Sets the core dump file size limit to -1, meaning there’s no restriction on the core dump size.
		//--ulimit nofile=65535:65535: Sets both the soft and hard limits for the number of open files to 65535.
		Out:         &containerId,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: TrimContainerId(&containerId),
	})
	t.AddStep(&step.ContainerExec{
		ContainerId: &containerId,
		Command:     fmt.Sprintf("bash %s/create_dingofs.sh %s %s %s %s"),
		Success:     &success,
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.StopContainer{
		ContainerId: containerId,
		ExecOptions: dingoadm.ExecOptions(),
	})

	return t, nil

}
