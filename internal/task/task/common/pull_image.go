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
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
)

func NewPullImageTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	hc, err := dingoadm.GetHost(dc.GetHost())
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s image=%s", dc.GetHost(), dc.GetContainerImage())
	t := task.NewTask("Pull Image", subname, hc.GetSSHConfig())

	// add step to task
	t.AddStep(&step.PullImage{
		Image:       dc.GetContainerImage(),
		ExecOptions: dingoadm.ExecOptions(),
	})

	return t, nil
}
