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
 * Created Date: 2021-11-26
 * Author: Jingli Chen (Wine93)
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

package common

import (
	"fmt"
	"path"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/internal/utils"
)

const (
	TEMP_DIR = "/tmp"
)

type (
	Step2CopyFilesFromContainer struct {
		Files         *[]string
		ContainerId   string
		HostDestDir   string
		ExcludeParent bool
		Dingoadm      *cli.DingoAdm
	}
)

func encodeSecret(secret string) string {
	return utils.MD5Sum(secret)
}

func (s *Step2CopyFilesFromContainer) Execute(ctx *context.Context) error {
	steps := []task.Step{}
	for _, file := range *s.Files {
		steps = append(steps, &step.CopyFromContainer{
			ContainerSrcPath: file,
			HostDestPath:     s.HostDestDir,
			ContainerId:      s.ContainerId,
			ExcludeParent:    s.ExcludeParent,
			ExecOptions:      s.Dingoadm.ExecOptions(),
		})
	}

	for _, step := range steps {
		err := step.Execute(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewCollectServiceTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	serviceId := dingoadm.GetServiceId(dc.GetId())
	containerId, err := dingoadm.Storage().GetContainerId(serviceId)
	if dingoadm.IsSkip(dc) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if len(containerId) == 0 {
		return nil, nil
	} else if containerId == comm.CLEANED_CONTAINER_ID {
		return nil, nil
	}
	hc, err := dingoadm.GetHost(dc.GetHost())
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s role=%s containerId=%s",
		dc.GetHost(), dc.GetRole(), tui.TrimContainerId(containerId))
	t := task.NewTask("Collect Service", subname, hc.GetSSHConfig())

	// add step to task
	var out string
	secret := dingoadm.MemStorage().Get(comm.KEY_SECRET).(string)
	urlFormat := dingoadm.MemStorage().Get(comm.KEY_SUPPORT_UPLOAD_URL_FORMAT).(string)
	baseDir := TEMP_DIR
	vname := utils.NewVariantName(fmt.Sprintf("%s_%s", serviceId, utils.RandString(5)))
	remoteSaveDir := fmt.Sprintf("%s/%s", baseDir, vname.Name)                // /tmp/7b510fb63730_ox1fe
	remoteTarbllPath := path.Join(baseDir, vname.CompressName)                // /tmp/7b510fb63730_ox1fe.tar.gz
	localTarballPath := path.Join(baseDir, vname.LocalCompressName)           // /tmp/7b510fb63730_ox1fe.local.tar.gz
	localEncryptdTarballPath := path.Join(baseDir, vname.EncryptCompressName) // /tmp/7b510fb63730_ox1fe-encrypted.tar.gz
	httpSavePath := path.Join("/", encodeSecret(secret), "service", dc.GetRole())
	layout := dc.GetProjectLayout()
	containerLogDir := layout.ServiceLogDir   // /curvebs/etcd/logs
	containerConfDir := layout.ServiceConfDir // /curvebs/etcd/conf
	localOptions := dingoadm.ExecOptions()
	localOptions.ExecInLocal = true

	t.AddStep(&step.CreateDirectory{
		Paths:       []string{remoteSaveDir},
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&Step2CopyFilesFromContainer{ // copy logs directory
		ContainerId: containerId,
		Files:       &[]string{containerLogDir},
		HostDestDir: remoteSaveDir,
		Dingoadm:    dingoadm,
	})
	t.AddStep(&Step2CopyFilesFromContainer{ // copy conf directory
		ContainerId: containerId,
		Files:       &[]string{containerConfDir},
		HostDestDir: remoteSaveDir,
		Dingoadm:    dingoadm,
	})
	t.AddStep(&step.ContainerLogs{
		ContainerId: containerId,
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.InstallFile{
		Content:      &out,
		HostDestPath: fmt.Sprintf("%s/docker.log", path.Join(remoteSaveDir, "logs")),
		ExecOptions:  dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Tar{
		File:        remoteSaveDir,
		Archive:     remoteTarbllPath,
		Create:      true,
		Gzip:        true,
		Verbose:     true,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.DownloadFile{
		RemotePath:  remoteTarbllPath,
		LocalPath:   localTarballPath,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step2EncryptFile{
		source: localTarballPath,
		dest:   localEncryptdTarballPath,
		secret: secret,
	})
	t.AddStep(&step.Curl{ // upload to curve team // curl -F "path=@$FILE" http://localhost:8080/upload\?path\=/
		Url:         fmt.Sprintf(urlFormat, httpSavePath),
		Form:        fmt.Sprintf("path=@%s", localEncryptdTarballPath),
		ExecOptions: localOptions,
	})
	t.AddPostStep(&step.RemoveFile{
		Files:       []string{remoteSaveDir, remoteTarbllPath},
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddPostStep(&step.RemoveFile{
		Files:       []string{localTarballPath, localEncryptdTarballPath},
		ExecOptions: localOptions,
	})

	return t, nil
}
