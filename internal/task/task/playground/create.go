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
 * Created Date: 2022-11-07
 * Author: Jingli Chen (Wine93)
 */

package playground

import (
	"fmt"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/pkg/module"
)

const (
	FORMAT_MOUNT_OPTION = "type=bind,source=%s,target=%s,bind-propagation=rshared"
)

type (
	step2CreateNBDDevice struct {
		execOptions module.ExecOptions
	}

	step2InsertPlayGround struct {
		curveadm *cli.DingoAdm
		cfg      *configure.PlaygroundConfig
	}
)

func getAttchMount(kind, mountPoint string) string {
	var mount string
	if kind == topology.KIND_CURVEBS {
		return mount
	}
	return fmt.Sprintf(FORMAT_MOUNT_OPTION, mountPoint, "/host")
}

func getMountVolumes(kind string) []step.Volume {
	volumes := []step.Volume{}
	if kind == topology.KIND_CURVEFS || kind == topology.KIND_DINGOFS {
		return volumes
	}

	return []step.Volume{
		{HostPath: "/dev", ContainerPath: "/dev"},
		{HostPath: "/lib/modules", ContainerPath: "/lib/modules"},
	}
}

func execOptions(curveadm *cli.DingoAdm) module.ExecOptions {
	options := curveadm.ExecOptions()
	options.ExecInLocal = true
	options.ExecWithSudo = false
	return options
}

func (s *step2CreateNBDDevice) Execute(ctx *context.Context) error {
	cmd := ctx.Module().Shell().ModProbe("nbd", "nbds_max=64")
	_, err := cmd.Execute(s.execOptions)
	return err
}

func (s *step2InsertPlayGround) Execute(ctx *context.Context) error {
	cfg := s.cfg
	err := s.curveadm.Storage().InsertPlayground(cfg.GetName(), cfg.GetMointpoint())
	if err != nil {
		return errno.ERR_INSERT_PLAYGROUND_FAILED.E(err)
	}
	return nil
}

func NewCreatePlaygroundTask(curveadm *cli.DingoAdm, cfg *configure.PlaygroundConfig) (*task.Task, error) {
	kind := cfg.GetKind()
	name := cfg.GetName()
	containerImage := cfg.GetContainIamge()
	mountPoint := cfg.GetMointpoint()

	// new task
	subname := fmt.Sprintf("kind=%s name=%s image=%s", kind, name, containerImage)
	t := task.NewTask("Create Playground", subname, nil)
	var containerId string

	// add step to task
	t.AddStep(&step2CreateNBDDevice{
		execOptions: execOptions(curveadm),
	})
	t.AddStep(&step.PullImage{
		Image:       containerImage,
		ExecOptions: execOptions(curveadm),
	})
	t.AddStep(&step.CreateContainer{
		Image:             containerImage,
		Envs:              []string{"LD_PRELOAD=/usr/local/lib/libjemalloc.so"},
		Entrypoint:        "/bin/bash",
		Command:           "/entrypoint.sh curvebs",
		Name:              name, // playground-curvebs-1656035414
		Network:           "bridge",
		Mount:             getAttchMount(kind, mountPoint),
		Volumes:           getMountVolumes(kind),
		Devices:           []string{"/dev/fuse"},
		SecurityOptions:   []string{"apparmor:unconfined"},
		LinuxCapabilities: []string{"SYS_ADMIN"},
		Ulimits:           []string{"core=-1"},
		Privileged:        true,
		Out:               &containerId,
		ExecOptions:       execOptions(curveadm),
	})
	t.AddStep(&step2InsertPlayGround{
		curveadm: curveadm,
		cfg:      cfg,
	})
	return t, nil
}
