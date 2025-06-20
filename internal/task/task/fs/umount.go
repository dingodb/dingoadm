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

package fs

import (
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
)

const (
	ONE_DAY_SECONDS       = 24 * 3600
	SIGNATURE_NOT_MOUNTED = "not mounted"
)

type (
	step2UmountFS struct {
		containerId string
		status      *string
		mountPoint  string
		dingoadm    *cli.DingoAdm
	}

	step2RemoveContainer struct {
		status      *string
		containerId string
		dingoadm    *cli.DingoAdm
	}

	step2DeleteClient struct {
		fsId     string
		dingoadm *cli.DingoAdm
	}
)

func (s *step2UmountFS) Execute(ctx *context.Context) error {
	if len(*s.status) == 0 || !strings.HasPrefix(*s.status, "Up") { // container already removed or not runing, remove it directly
		cmd := ctx.Module().Shell().Umount(s.mountPoint)
		cmd.Execute(s.dingoadm.ExecOptions())
		return nil
	}

	command := fmt.Sprintf("umount %s", configure.GetFSClientMountPath(s.mountPoint))
	dockerCli := ctx.Module().DockerCli().ContainerExec(s.containerId, command)
	out, err := dockerCli.Execute(s.dingoadm.ExecOptions())
	if strings.Contains(out, SIGNATURE_NOT_MOUNTED) {
		return nil
	} else if err == nil {
		return nil
	}
	return errno.ERR_UMOUNT_FILESYSTEM_FAILED.S(out)
}

func (s *step2DeleteClient) Execute(ctx *context.Context) error {
	err := s.dingoadm.Storage().DeleteClient(s.fsId)
	if err != nil {
		return errno.ERR_DELETE_CLIENT_FAILED.E(err)
	}

	err = s.dingoadm.Storage().DeleteClientConfig(s.fsId)
	if err != nil {
		return errno.ERR_DELETE_CLIENT_CONFIG_FAILED.E(err)
	}

	return nil
}

func (s *step2RemoveContainer) Execute(ctx *context.Context) error {
	if len(*s.status) == 0 {
		return nil
	}

	steps := []task.Step{}
	options := s.dingoadm.ExecOptions()
	options.ExecTimeoutSec = ONE_DAY_SECONDS // wait all data flushed to S3
	if strings.HasPrefix(*s.status, "Up") {
		// stop container
		steps = append(steps, &step.StopContainer{
			ContainerId: s.containerId,
			ExecOptions: options,
		})
		// wait container stop
		steps = append(steps, &step.WaitContainer{
			ContainerId: s.containerId,
			ExecOptions: options,
		})
	}
	steps = append(steps, &step.RemoveContainer{
		ContainerId: s.containerId,
		ExecOptions: s.dingoadm.ExecOptions(),
	})

	for _, step := range steps {
		err := step.Execute(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewUmountFSTask(dingoadm *cli.DingoAdm, v interface{}) (*task.Task, error) {
	options := dingoadm.MemStorage().Get(comm.KEY_MOUNT_OPTIONS).(MountOptions)
	fsId := dingoadm.GetFilesystemId(options.Host, options.MountPoint)
	hc, err := dingoadm.GetHost(options.Host)
	if err != nil {
		return nil, err
	}

	// new task
	mountPoint := options.MountPoint
	subname := fmt.Sprintf("host=%s mountPoint=%s", options.Host, mountPoint)
	t := task.NewTask("Umount FileSystem", subname, hc.GetSSHConfig())

	// add step to task
	var status string
	containerId := mountPoint2ContainerName(mountPoint)

	t.AddStep(&step.ListContainers{
		ShowAll:     true,
		Format:      "'{{.Status}}'",
		Filter:      fmt.Sprintf("name=%s", containerId),
		Out:         &status,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step2UmountFS{
		containerId: containerId,
		status:      &status,
		mountPoint:  options.MountPoint,
		dingoadm:    dingoadm,
	})
	t.AddStep(&step2RemoveContainer{
		status:      &status,
		containerId: containerId,
		dingoadm:    dingoadm,
	})
	t.AddStep(&step2DeleteClient{
		dingoadm: dingoadm,
		fsId:     fsId,
	})

	return t, nil
}
