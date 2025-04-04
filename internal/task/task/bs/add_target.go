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
 * Created Date: 2022-02-08
 * Author: Jingli Chen (Wine93)
 */

package bs

import (
	"fmt"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/task/scripts"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
)

type TargetOption struct {
	Host      string
	User      string
	Volume    string
	Create    bool
	Size      int
	Tid       string
	Blocksize uint64
}

func NewAddTargetTask(curveadm *cli.DingoAdm, cc *configure.ClientConfig) (*task.Task, error) {
	options := curveadm.MemStorage().Get(comm.KEY_TARGET_OPTIONS).(TargetOption)
	user, volume := options.User, options.Volume
	hc, err := curveadm.GetHost(options.Host)
	if err != nil {
		return nil, err
	}

	subname := fmt.Sprintf("host=%s volume=%s", options.Host, volume)
	t := task.NewTask("Add Target", subname, hc.GetSSHConfig())

	// add step
	var output string
	containerId := DEFAULT_TGTD_CONTAINER_NAME
	targetScriptPath := "/curvebs/tools/sbin/target.sh"
	targetScript := scripts.TARGET
	cmd := fmt.Sprintf("/bin/bash %s %s %s %v %d %d", targetScriptPath, user, volume, options.Create, options.Size, options.Blocksize)
	toolsConf := fmt.Sprintf(FORMAT_TOOLS_CONF, cc.GetClusterMDSAddr())

	t.AddStep(&step.ListContainers{
		ShowAll:     true,
		Format:      "'{{.ID}} {{.Status}}'",
		Filter:      fmt.Sprintf("name=%s", DEFAULT_TGTD_CONTAINER_NAME),
		Out:         &output,
		ExecOptions: curveadm.ExecOptions(),
	})
	t.AddStep(&step2CheckTgtdStatus{
		output: &output,
	})
	t.AddStep(&step.InstallFile{ // install tools.conf
		Content:           &toolsConf,
		ContainerId:       &containerId,
		ContainerDestPath: "/etc/dingo/tools.conf",
		ExecOptions:       curveadm.ExecOptions(),
	})
	t.AddStep(&step.InstallFile{ // install target.sh
		Content:           &targetScript,
		ContainerId:       &containerId,
		ContainerDestPath: targetScriptPath,
		ExecOptions:       curveadm.ExecOptions(),
	})
	t.AddStep(&step.ContainerExec{
		ContainerId: &containerId,
		Command:     cmd,
		ExecOptions: curveadm.ExecOptions(),
	})

	return t, nil
}
