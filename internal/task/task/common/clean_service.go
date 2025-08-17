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
	"strconv"
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/storage"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/scripts"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/internal/task/task/bs"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/internal/utils"
	"github.com/dingodb/dingoadm/pkg/module"
)

const (
	LAYOUT_CURVEBS_CHUNKFILE_POOL_DIR = topology.LAYOUT_CURVEBS_CHUNKFILE_POOL_DIR
	LAYOUT_CURVEBS_COPYSETS_DIR       = topology.LAYOUT_CURVEBS_COPYSETS_DIR
	LAYOUT_CURVEBS_RECYCLER_DIR       = topology.LAYOUT_CURVEBS_RECYCLER_DIR
	METAFILE_CHUNKSERVER_ID           = topology.METAFILE_CHUNKSERVER_ID

	SIGNATURE_CONTAINER_REMOVED = "No such container"
)

type (
	step2RecycleChunk struct {
		dc                *topology.DeployConfig
		clean             map[string]bool
		recycleScriptPath string
		execOptions       module.ExecOptions
	}

	Step2CleanContainer struct {
		ServiceId   string
		ContainerId string
		Storage     *storage.Storage
		ExecOptions module.ExecOptions
	}
)

func (s *step2RecycleChunk) Execute(ctx *context.Context) error {
	dc := s.dc
	if !s.clean[comm.CLEAN_ITEM_DATA] {
		return nil
	} else if dc.GetRole() != topology.ROLE_CHUNKSERVER {
		return nil
	} else if len(dc.GetDataDir()) == 0 {
		return nil
	}

	dataDir := dc.GetDataDir()
	copysetsDir := fmt.Sprintf("%s/%s", dataDir, LAYOUT_CURVEBS_COPYSETS_DIR)
	recyclerDir := fmt.Sprintf("%s/%s", dataDir, LAYOUT_CURVEBS_RECYCLER_DIR)
	source := fmt.Sprintf("'%s %s'", copysetsDir, recyclerDir)
	dest := fmt.Sprintf("%s/%s", dataDir, LAYOUT_CURVEBS_CHUNKFILE_POOL_DIR)
	chunkSize := strconv.Itoa(bs.DEFAULT_CHUNKFILE_SIZE + bs.DEFAULT_CHUNKFILE_HEADER_SIZE)
	cmd := ctx.Module().Shell().BashScript(s.recycleScriptPath, source, dest, chunkSize)
	_, err := cmd.Execute(s.execOptions)
	if err != nil {
		errno.ERR_RUN_SCRIPT_FAILED.E(err)
	}
	return nil
}

func (s *Step2CleanContainer) Execute(ctx *context.Context) error {
	containerId := s.ContainerId
	if len(containerId) == 0 { // container not created
		return nil
	} else if containerId == comm.CLEANED_CONTAINER_ID { // container has removed
		return nil
	}

	cli := ctx.Module().DockerCli().RemoveContainer(s.ContainerId)
	cli.Execute(s.ExecOptions)
	//out, err := cli.Execute(s.ExecOptions)

	// container has removed
	//if err != nil && !strings.Contains(out, SIGNATURE_CONTAINER_REMOVED) {
	//	// fmt.Printf("container removed error: %s", out)
	//	return err
	//}
	return s.Storage.SetContainId(s.ServiceId, comm.CLEANED_CONTAINER_ID)
}

func getCleanFiles(clean map[string]bool, dc *topology.DeployConfig, recycle bool) []string {
	files := []string{}
	for item := range clean {
		switch item {
		case comm.CLEAN_ITEM_LOG:
			files = append(files, dc.GetLogDir())
		case comm.CLEAN_ITEM_DATA:
			if dc.GetRole() != topology.ROLE_CHUNKSERVER || !recycle {
				files = append(files, dc.GetDataDir())
			} else {
				dataDir := dc.GetDataDir()
				copysetsDir := fmt.Sprintf("%s/%s", dataDir, LAYOUT_CURVEBS_COPYSETS_DIR)
				chunkserverIdMetafile := fmt.Sprintf("%s/%s", dataDir, METAFILE_CHUNKSERVER_ID)
				files = append(files, copysetsDir, chunkserverIdMetafile)
			}
		case comm.CLEAN_ITEM_RAFT:
			if dc.GetRole() == topology.ROLE_COORDINATOR || dc.GetRole() == topology.ROLE_STORE {
				files = append(files, dc.GetDingoRaftDir())
			}
		}
	}
	return files
}

func NewCleanServiceTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
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

	if dc.GetRole() == topology.ROLE_MDSV2_CLI {
		skipTmp := dingoadm.MemStorage().Get(comm.KEY_SKIP_MDSV2_CLI)
		if skipTmp != nil && skipTmp.(bool) {
			return nil, nil
		}
	}

	// new task
	only := dingoadm.MemStorage().Get(comm.KEY_CLEAN_ITEMS).([]string)
	recycle := dingoadm.MemStorage().Get(comm.KEY_CLEAN_BY_RECYCLE).(bool)
	subname := fmt.Sprintf("host=%s role=%s containerId=%s clean=%s",
		dc.GetHost(), dc.GetRole(), tui.TrimContainerId(containerId), strings.Join(only, ","))
	t := task.NewTask("Clean Service", subname, hc.GetSSHConfig())

	// add step to task
	clean := utils.Slice2Map(only)
	files := getCleanFiles(clean, dc, recycle) // directorys which need cleaned
	recyleScript := scripts.RECYCLE
	recyleScriptPath := utils.RandFilename(TEMP_DIR)

	if dc.GetKind() == topology.KIND_CURVEBS {
		t.AddStep((&step.InstallFile{
			Content:      &recyleScript,
			HostDestPath: recyleScriptPath,
			ExecOptions:  dingoadm.ExecOptions(),
		}))
		t.AddStep(&step2RecycleChunk{
			dc:                dc,
			clean:             clean,
			recycleScriptPath: recyleScriptPath,
			execOptions:       dingoadm.ExecOptions(),
		})
	}
	t.AddStep(&step.RemoveFile{
		Files:       files,
		ExecOptions: dingoadm.ExecOptions(),
	})
	if clean[comm.CLEAN_ITEM_CONTAINER] == true {

		// var status string
		// t.AddStep(&step.InspectContainer{
		// ContainerId: containerId,
		// Format:      "'{{.State.Status}}'",
		// Out:         &status,
		// ExecOptions: dingoadm.ExecOptions(),
		// })
		// if err != nil {
		// return errno.ERR_CONTAINER_NOT_EXISTED.S(*s.Out)
		// } else if status != "running" {
		// return errno.ERR_CONTAINER_IS_ABNORMAL.
		// F("host=%s role=%s containerId=%s",
		// s.Host, s.Role, tui.TrimContainerId(s.ContainerId))
		// }
		// return nil

		t.AddStep(&Step2CleanContainer{
			ServiceId:   serviceId,
			ContainerId: containerId,
			Storage:     dingoadm.Storage(),
			ExecOptions: dingoadm.ExecOptions(),
		})
	}

	return t, nil
}
