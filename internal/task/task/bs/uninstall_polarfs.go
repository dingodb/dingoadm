/*
 *  Copyright (c) 2022 NetEase Inc.
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
 * Created Date: 2022-08-08
 * Author: Jingli Chen (Wine93)
 */

package bs

import (
	"fmt"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
)

type step2UninstallPackage struct {
	release  string
	curveadm *cli.DingoAdm
}

func (s *step2UninstallPackage) Execute(ctx *context.Context) error {
	steps := []task.Step{}
	curveadm := s.curveadm
	release := s.release
	if release == comm.OS_RELEASE_DEBIAN ||
		release == comm.OS_RELEASE_UBUNTU {
		steps = append(steps, &step.Dpkg{
			Purge:       "pfsd",
			ExecOptions: curveadm.ExecOptions(),
		})
	} else if release == comm.OS_RELEASE_CENTOS {
		steps = append(steps, &step.Rpm{
			// do something
			ExecOptions: curveadm.ExecOptions(),
		})
	} else {
		return errno.ERR_UNSUPPORT_LINUX_OS_REELASE.
			F("os release: %s", release)
	}

	for _, step := range steps {
		err := step.Execute(ctx)
		if err != nil {
			return errno.ERR_INSTALL_PFSD_PACKAGE_FAILED.E(err)
		}
	}
	return nil
}

func NewUninstallPolarFSTask(curveadm *cli.DingoAdm, v interface{}) (*task.Task, error) {
	host := curveadm.MemStorage().Get(comm.KEY_POLARFS_HOST).(string)
	hc, err := curveadm.GetHost(host)
	if err != nil {
		return nil, err
	}

	// new task
	release := getRelease(curveadm)
	subname := fmt.Sprintf("host=%s release=%s", host, release)
	t := task.NewTask("Uninstall PolarFS", subname, hc.GetSSHConfig())

	// add step to task
	t.AddStep(&step.RemoveFile{
		Files:       []string{"/etc/dingo"},
		ExecOptions: curveadm.ExecOptions(),
	})
	t.AddPostStep(&step2UninstallPackage{
		release:  release,
		curveadm: curveadm,
	})

	return t, nil
}
