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
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/build"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/storage"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/scripts"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/internal/utils"
)

type step2SetClusterPool struct {
	dingoadm    *cli.DingoAdm
	clusterPool string
	storage     *storage.Storage
}

func getPoolset(dingoadm *cli.DingoAdm, kind string) configure.Poolset {
	if kind == configure.KIND_DINGOFS {
		return configure.Poolset{}
	}
	return dingoadm.MemStorage().Get(comm.KEY_POOLSET).(configure.Poolset)
}

func getClusterPool(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (configure.DingoFsClusterTopo, error) {
	poolset := getPoolset(dingoadm, dc.GetKind())
	oldPool := configure.DingoFsClusterTopo{}
	dcs, err := dingoadm.ParseTopology()
	if err != nil {
		return oldPool, err
	}

	// 1) generate a new default pool
	data := dingoadm.ClusterPoolData()
	if len(data) == 0 {
		return configure.GenerateDefaultClusterPool(dcs, poolset)
	}

	// 2) OR change old pool and return it
	err = json.Unmarshal([]byte(data), &oldPool)
	if err != nil {
		return oldPool, err
	}
	pool, err := configure.GenerateDefaultClusterPool(dcs, poolset)
	if err != nil {
		return pool, err
	}

	// NOTE: curveadm gurantee oldPool and pool has same servers
	for i, server := range pool.Servers {
		oldPool.Servers[i].InternalIp = server.InternalIp
		oldPool.Servers[i].InternalPort = server.InternalPort
		oldPool.Servers[i].ExternalIp = server.ExternalIp
		oldPool.Servers[i].ExternalPort = server.ExternalPort
	}
	if dc.GetKind() == topology.KIND_CURVEBS {
		for i, pool := range pool.LogicalPools {
			oldPool.LogicalPools[i].Copysets = pool.Copysets
		}
	}

	return oldPool, err
}

func prepare(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (clusterPoolJson, clusterMDSAddrs string, err error) {
	// 1. get origin cluster pool
	var clusterPool configure.DingoFsClusterTopo
	clusterPool, err = getClusterPool(dingoadm, dc)
	if err != nil {
		return
	}

	// 2. scale out cluster or migrate servers
	if dingoadm.MemStorage().Get(comm.KEY_SCALE_OUT_CLUSTER) != nil { // scale out cluster
		dcs := dingoadm.MemStorage().Get(comm.KEY_SCALE_OUT_CLUSTER).([]*topology.DeployConfig)
		poolset := getPoolset(dingoadm, dc.GetKind())
		configure.ScaleOutClusterPool(&clusterPool, dcs, poolset)
	} else if dingoadm.MemStorage().Get(comm.KEY_MIGRATE_SERVERS) != nil { // migrate servers
		migrates := dingoadm.MemStorage().Get(comm.KEY_MIGRATE_SERVERS).([]*configure.MigrateServer)
		configure.MigrateClusterServer(&clusterPool, migrates)
	}

	// 3. encode cluster pool to json string
	var bytes []byte
	bytes, err = json.Marshal(clusterPool)
	if err != nil {
		return
	}
	clusterPoolJson = string(bytes)

	// cluster MDS address
	clusterMDSAddrs, err = dc.GetVariables().Get("cluster_mds_addr")
	clusterMDSAddrs = strings.Replace(clusterMDSAddrs, ",", " ", -1)
	return
}

func checkWaitMDSElectionSuccess(success *bool, out *string) step.LambdaType {
	return func(ctx *context.Context) error {
		if !*success {
			return errno.ERR_WAIT_MDS_ELECTION_SUCCESS_TIMEOUT
		}
		return nil
	}
}

func checkChunkserverOnline(success *bool, out *string) step.LambdaType {
	return func(ctx *context.Context) error {
		if !*success {
			return errno.ERR_WAIT_ALL_CHUNKSERVERS_ONLINE_TIMEOUT
		}
		return nil
	}
}

func genCreatePoolCommand(dc *topology.DeployConfig, pooltype, poolJSONPath string) string {
	layout := dc.GetProjectLayout()
	toolsBinaryPath := layout.ToolsV2BinaryPath // v1: ToolsBinaryPath , v2: ToolsV2BinaryPath
	if dc.GetKind() == topology.KIND_DINGOFS {
		// for curvefs, the default topology json path is current directory's topology.json
		return fmt.Sprintf("%s create topology --clustermap=%s", toolsBinaryPath, poolJSONPath)
	}
	// v1
	return fmt.Sprintf("%s -op=create_%s -cluster_map=%s",
		toolsBinaryPath, pooltype, poolJSONPath)
}

func checkCreatePoolStatus(success *bool, out *string) step.LambdaType {
	return func(ctx *context.Context) error {
		if !*success {
			return errno.ERR_CREATE_LOGICAL_POOL_FAILED.S(*out)
		}
		return nil
	}
}

func (s *step2SetClusterPool) Execute(ctx *context.Context) error {
	dingoadm := s.dingoadm
	topology := dingoadm.ClusterTopologyData()
	value := dingoadm.MemStorage().Get(comm.KEY_NEW_TOPOLOGY_DATA)
	if value != nil {
		topology = value.(string)
	}

	err := s.storage.SetClusterPool(dingoadm.ClusterId(), topology, s.clusterPool)
	if err != nil {
		return errno.ERR_UPDATE_CLUSTER_POOL_FAILED.E(err)
	}
	return nil
}

func NewCreateTopologyTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
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
	pooltype := dingoadm.MemStorage().Get(comm.KEY_CREATE_POOL_TYPE).(string)
	name := utils.Choose(pooltype == comm.POOL_TYPE_LOGICAL,
		"Create Logical Pool", "Create Physical Pool")
	subname := fmt.Sprintf("host=%s role=%s containerId=%s",
		dc.GetHost(), dc.GetRole(), tui.TrimContainerId(containerId))
	t := task.NewTask(name, subname, hc.GetSSHConfig())

	// add step to task
	var success bool
	var out string
	host, role := dc.GetHost(), dc.GetRole()
	layout := dc.GetProjectLayout()
	poolJSONPath := fmt.Sprintf("%s/topology.json", layout.ToolsV2ConfDir) // v1: ToolsConfDir , v2: ToolsV2ConfDir
	waitScript := scripts.WAIT
	waitScriptPath := fmt.Sprintf("%s/wait.sh", layout.ToolsV2BinDir) // v1: ToolsBinDir, v2: ToolsV2BinDir
	clusterPoolJson, clusterMDSAddrs, err := prepare(dingoadm, dc)
	if err != nil {
		return nil, err
	}
	build.DEBUG(build.DEBUG_CREATE_POOL,
		build.Field{"pool json", clusterPoolJson})

	t.AddStep(&step.ListContainers{
		ShowAll:     true,
		Format:      `"{{.ID}}"`,
		Filter:      fmt.Sprintf("id=%s", containerId),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: CheckContainerExist(host, role, containerId, &out),
	})
	t.AddStep(&step.InstallFile{ // install curvebs/curvefs topology
		ContainerId:       &containerId,
		ContainerDestPath: poolJSONPath,
		Content:           &clusterPoolJson,
		ExecOptions:       dingoadm.ExecOptions(),
	})
	t.AddStep(&step.InstallFile{ // install wait script
		ContainerId:       &containerId,
		ContainerDestPath: waitScriptPath,
		Content:           &waitScript,
		ExecOptions:       dingoadm.ExecOptions(),
	})
	t.AddStep(&step.ContainerExec{ // wait mds leader election success
		ContainerId: &containerId,
		Command:     fmt.Sprintf("bash %s %s", waitScriptPath, clusterMDSAddrs),
		Success:     &success,
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: checkWaitMDSElectionSuccess(&success, &out),
	})

	if dc.GetKind() == topology.KIND_CURVEBS && pooltype == comm.POOL_TYPE_LOGICAL {
		waitChunkserversScript := scripts.WAIT_CHUNKSERVERS
		waitChunkserversScriptPath := fmt.Sprintf("%s/wait_chunkservers.sh", layout.ToolsBinDir)
		nchunkserver := dingoadm.MemStorage().Get(comm.KEY_NUMBER_OF_CHUNKSERVER).(int)
		t.AddStep(&step.InstallFile{ // install wait_chunkservers script
			ContainerId:       &containerId,
			ContainerDestPath: waitChunkserversScriptPath,
			Content:           &waitChunkserversScript,
			ExecOptions:       dingoadm.ExecOptions(),
		})
		t.AddStep(&step.ContainerExec{ // wait all chunkservers online before create logical pool
			ContainerId: &containerId,
			Command:     fmt.Sprintf("bash %s %d", waitChunkserversScriptPath, nchunkserver),
			Success:     &success,
			Out:         &out,
			ExecOptions: dingoadm.ExecOptions(),
		})
		t.AddStep(&step.Lambda{
			Lambda: checkChunkserverOnline(&success, &out),
		})
	}
	t.AddStep(&step.ContainerExec{ // create topology
		ContainerId: &containerId,
		Success:     &success,
		Out:         &out,
		Command:     genCreatePoolCommand(dc, pooltype, poolJSONPath),
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: checkCreatePoolStatus(&success, &out),
	})
	if pooltype == comm.POOL_TYPE_LOGICAL {
		t.AddStep(&step2SetClusterPool{
			dingoadm:    dingoadm,
			clusterPool: clusterPoolJson,
			storage:     dingoadm.Storage(),
		})
	}

	return t, nil
}
