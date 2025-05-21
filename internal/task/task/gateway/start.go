/*
 * Copyright (c) 2024 dingodb.com, Inc. All Rights Reserved
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * Project: DingoFS
 * Created Date: 2024-10-28
 * Author: Wei Dong (jackblack369)
 */

package gateway

import (
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/internal/configure"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/scripts"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/internal/task/task/checker"
	"github.com/dingodb/dingoadm/internal/utils"
)

func NewStartGatewayTask(curveadm *cli.DingoAdm, gc *configure.GatewayConfig) (*task.Task, error) {
	host := curveadm.MemStorage().Get(comm.GATEWAY_HOST).(string)
	hc, err := curveadm.GetHost(host)
	if err != nil {
		return nil, err
	}

	// new task
	mdsaddr := curveadm.MemStorage().Get(comm.MDSADDR).(string)
	gatewayListenAddr := curveadm.MemStorage().Get(comm.GATEWAY_LISTEN_ADDR).(string)
	gatewayConsoleAddr := curveadm.MemStorage().Get(comm.GATEWAY_CONSOLE_ADDR).(string)
	mountPoint := curveadm.MemStorage().Get(comm.GATEWAY_MOUNTPOINT).(string)

	subname := fmt.Sprintf("gatewayListenAddr=%s gatewayConsoleAddr=%s mountPoint=%s", gatewayListenAddr, gatewayConsoleAddr, mountPoint)
	t := task.NewTask("Bootstrap S3 ateway service", subname, hc.GetSSHConfig())

	// add step to task
	var containerId, out string
	var success bool
	containerName := fmt.Sprintf("dingofs-gateway-%s", utils.MD5Sum(mountPoint))
	containerMountPath := fmt.Sprintf("%s/client/mnt%s", topology.GetDingoFSProjectLayout().ProjectRootDir, mountPoint)
	startGatewayScript := scripts.START_GATEWAY
	startGatewayScriptPath := "/gateway.sh"

	t.AddStep(&step.EngineInfo{
		Success:     &success,
		Out:         &out,
		ExecOptions: curveadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: checker.CheckEngineInfo(host, curveadm.ExecOptions().ExecWithEngine, &success, &out),
	})

	// todo add check same gateway service have existed

	t.AddStep(&step.PullImage{
		Image:       gc.GetContainerImage(),
		ExecOptions: curveadm.ExecOptions(),
	})
	t.AddStep(&step.CreateContainer{
		Image:      gc.GetContainerImage(),
		Command:    getStartGatewayCommand(mdsaddr, gatewayListenAddr, gatewayConsoleAddr, containerMountPath),
		Entrypoint: "/bin/bash",
		Envs:       getEnvironments(gc),
		Init:       true,
		Name:       containerName,
		Mount:      fmt.Sprintf("type=bind,source=%s,target=%s,bind-propagation=rshared", mountPoint, containerMountPath),
		//Volumes:           getMountVolumes(cc),
		Ulimits:     []string{"core=-1", "nofile=65535:65535"},
		Out:         &containerId,
		ExecOptions: curveadm.ExecOptions(),
	})

	//todo save gateway info to db

	t.AddStep(&step.InstallFile{ // install gateway.sh shell
		ContainerId:       &containerId,
		ContainerDestPath: startGatewayScriptPath,
		Content:           &startGatewayScript,
		ExecOptions:       curveadm.ExecOptions(),
	})

	t.AddStep(&step.StartContainer{
		ContainerId: &containerId,
		Success:     &success,
		Out:         &out,
		ExecOptions: curveadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: checkStartContainerStatus(&success, &out),
	})

	return t, nil

}

func getStartGatewayCommand(mdsaddr string, gatewayListenAddr string, gatewayConsoleAddr string, mountPoint string) string {
	return fmt.Sprintf("/gateway.sh %s %s %s %s", mdsaddr, gatewayListenAddr, gatewayConsoleAddr, mountPoint)
}

func getEnvironments(gc *configure.GatewayConfig) []string {
	envs := []string{
		"LD_PRELOAD=/usr/local/lib/libjemalloc.so",
		fmt.Sprintf("MINIO_ROOT_USER=%s", gc.GetS3RootUser()),
		fmt.Sprintf("MINIO_ROOT_PASSWORD=%s", gc.GetS3RootPassword()),
	}

	return envs
}

func checkStartContainerStatus(success *bool, out *string) step.LambdaType {
	return func(ctx *context.Context) error {
		if *success {
			return nil
		} else if strings.Contains(*out, "START GATEWAY FAILED") {
			return errno.ERR_START_GATEWAY_FAILED
		}
		return errno.ERR_START_GATEWAY_FAILED.S(*out)
	}
}
