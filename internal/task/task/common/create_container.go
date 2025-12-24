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
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	log "github.com/dingodb/dingoadm/pkg/log/glg"
)

const (
	POLICY_ALWAYS_RESTART                        = "always"
	POLICY_NEVER_RESTART                         = "no"
	ENV_DINGOSTORE_SERVER_LISTEN_HOST            = "SERVER_LISTEN_HOST"
	ENV_DINGOSTORE_RAFT_LISTEN_HOST              = "RAFT_LISTEN_HOST"
	ENV_DINGO_SERVER_LISTEN_HOST                 = "SERVER_LISTEN_HOST"
	ENV_DINGO_SERVER_HOST                        = "SERVER_HOST"
	ENV_DINGOSTORE_RAFT_HOST                     = "RAFT_HOST"
	ENV_DINGOSTORE_DEFAULT_REPLICA_NUM           = "DEFAULT_REPLICA_NUM"
	ENV_DINGOSTORE_COOR_RAFT_PEERS               = "COOR_RAFT_PEERS"
	ENV_DINGOSTORE_COOR_SRV_PEERS                = "COOR_SRV_PEERS"
	ENV_DINGOSTROE_FLAGS_ROLE                    = "FLAGS_role"
	ENV_DINGOSTROE_FLAGS_CLEAN_LOG               = "FLAGS_clean_log"
	ENV_DINGOSTORE_COORDINATOR_SERVER_START_PORT = "COORDINATOR_SERVER_START_PORT"
	ENV_DINGOSTORE_COORDINATOR_RAFT_START_PORT   = "COORDINATOR_RAFT_START_PORT"
	ENV_DINGO_SERVER_START_PORT                  = "SERVER_START_PORT"
	ENV_DINGOSTORE_RAFT_START_PORT               = "RAFT_START_PORT"
	ENV_DINGOSTORE_INSTANCE_START_ID             = "INSTANCE_START_ID"
	ENV_DINGOSTORE_ENABLE_LITE                   = "ENABLE_LITE"

	// dingodb
	ENV_DINGOSTORE_DISKANN_SERVER_HOST       = "DISKANN_SERVER_HOST"
	ENV_DINGOSTORE_DISKANN_SERVER_START_PORT = "DISKANN_SERVER_START_PORT"
	ENV_DINGOSTORE_DISKANN_INSTANCE_START_ID = "DISKANN_INSTANCE_START_ID"
	ENV_DINGOSTORE_INDEX_SERVER_START_PORT   = "INDEX_SERVER_START_PORT"
	ENV_DINGOSTORE_INDEX_RAFT_START_PORT     = "INDEX_RAFT_START_PORT"
	ENV_DINGOSTORE_INDEX_INSTANCE_START_ID   = "INDEX_INSTANCE_START_ID"

	// dingofs mdsv2
	ENV_DINGOFS_V2_FLAGS_ROLE        = "FLAGS_role"
	ENV_DINGOFS_V2_FLAGS_CLEAN_LOG   = "FLAGS_clean_log"
	ENV_DINGOFS_V2_FLAGS_SERVER_NUM  = "FLAGS_server_num"
	ENV_DINGOSTORE_COORDINATOR_ADDR  = "COORDINATOR_ADDR"
	ENV_DINGOFS_V2_INSTANCE_START_ID = "MDS_INSTANCE_START_ID"

	// dingodb executor
	ENV_DINGODB_EXECUTOR_ROLE         = "DINGO_ROLE"
	ENV_DINGODB_EXECUTOR_HOSTNAME     = "DINGO_HOSTNAME"
	ENV_DINGODB_EXECUTOR_COORDINATORS = "DINGO_COORDINATORS"
)

type Step2GetService struct {
	ServiceId   string
	ContainerId *string
	Storage     *storage.Storage
}

type Step2InsertService struct {
	ClusterId      int
	ServiceId      string
	ContainerId    *string
	OldContainerId *string
	Storage        *storage.Storage
}

func (s *Step2GetService) Execute(ctx *context.Context) error {
	containerId, err := s.Storage.GetContainerId(s.ServiceId)

	if err != nil {
		return errno.ERR_GET_SERVICE_CONTAINER_ID_FAILED.E(err)
	} else if containerId == comm.CLEANED_CONTAINER_ID { // "-" means container removed
		// do nothing
	} else if len(containerId) > 0 {
		return task.ERR_SKIP_TASK
	}

	*s.ContainerId = containerId
	return nil
}

func (s *Step2InsertService) E(e error, ec *errno.ErrorCode) error {
	if e == nil {
		return nil
	}
	return ec.E(e)
}

func (s *Step2InsertService) Execute(ctx *context.Context) error {
	var err error
	serviceId := s.ServiceId
	clusterId := s.ClusterId
	oldContainerId := *s.OldContainerId
	containerId := *s.ContainerId
	if oldContainerId == comm.CLEANED_CONTAINER_ID { // container cleaned
		err = s.Storage.SetContainId(serviceId, containerId)
		err = s.E(err, errno.ERR_SET_SERVICE_CONTAINER_ID_FAILED)
	} else {
		err = s.Storage.InsertService(clusterId, serviceId, containerId)
		err = s.E(err, errno.ERR_INSERT_SERVICE_CONTAINER_ID_FAILED)
	}

	log.SwitchLevel(err)("Insert service",
		log.Field("ServiceId", serviceId),
		log.Field("ContainerId", containerId))

	return err
}

func getArguments(dc *topology.DeployConfig) string {
	role := dc.GetRole()
	if role != topology.ROLE_CHUNKSERVER {
		return ""
	}

	// only chunkserver need so many arguments, but who cares
	layout := dc.GetProjectLayout()
	dataDir := layout.ServiceDataDir
	chunkserverArguments := map[string]interface{}{
		// chunkserver
		"conf":                  layout.ServiceConfPath,
		"chunkServerIp":         dc.GetListenIp(),
		"enableExternalServer":  dc.GetEnableExternalServer(),
		"chunkServerExternalIp": dc.GetListenExternalIp(),
		"chunkServerPort":       dc.GetListenPort(),
		"chunkFilePoolDir":      dataDir,
		"chunkFilePoolMetaPath": fmt.Sprintf("%s/chunkfilepool.meta", dataDir),
		"walFilePoolDir":        dataDir,
		"walFilePoolMetaPath":   fmt.Sprintf("%s/walfilepool.meta", dataDir),
		"copySetUri":            fmt.Sprintf("local://%s/copysets", dataDir),
		"recycleUri":            fmt.Sprintf("local://%s/recycler", dataDir),
		"raftLogUri":            fmt.Sprintf("curve://%s/copysets", dataDir),
		"raftSnapshotUri":       fmt.Sprintf("curve://%s/copysets", dataDir),
		"chunkServerStoreUri":   fmt.Sprintf("local://%s", dataDir),
		"chunkServerMetaUri":    fmt.Sprintf("local://%s/chunkserver.dat", dataDir),
		// brpc
		"bthread_concurrency":      18,
		"graceful_quit_on_sigterm": true,
		// raft
		"raft_sync":                            true,
		"raft_sync_meta":                       true,
		"raft_sync_segments":                   true,
		"raft_max_segment_size":                8388608,
		"raft_max_install_snapshot_tasks_num":  1,
		"raft_use_fsync_rather_than_fdatasync": false,
	}

	arguments := []string{}
	for k, v := range chunkserverArguments {
		arguments = append(arguments, fmt.Sprintf("-%s=%v", k, v))
	}
	return strings.Join(arguments, " ")
}

func getContainerCMD(dc *topology.DeployConfig) string {
	//upgrade_flag := dingoadm.MemStorage().Get(comm.KEY_UPGRADE_FLAG).(bool)
	cmd := "deploystart" // cleanstart
	//if upgrade_flag {
	//	cmd = "deploystart"
	//}
	switch dc.GetKind() {
	case topology.KIND_DINGOFS:
		if dc.GetRole() == topology.ROLE_DINGODB_EXECUTOR {
			return ""
		} else if dc.GetRole() == topology.ROLE_COORDINATOR || dc.GetRole() == topology.ROLE_STORE {
			// coordinator and store use cleanstart
			return cmd
		} else if dc.GetRole() == topology.ROLE_FS_MDS && dc.GetCtx().Lookup(topology.CTX_KEY_MDS_VERSION) == topology.CTX_VAL_MDS_V2 {
			return ""
		} else {
			return fmt.Sprintf("--role %s --args='%s'", dc.GetRole(), getArguments(dc))
		}
	case topology.KIND_DINGOSTORE, topology.KIND_DINGODB:
		if dc.GetRole() == topology.ROLE_DINGODB_EXECUTOR || dc.GetRole() == topology.ROLE_DINGODB_WEB || dc.GetRole() == topology.ROLE_DINGODB_PROXY {
			return ""
		} else {
			return cmd
		}
	default:
		return fmt.Sprintf("--role %s --args='%s'", dc.GetRole(), getArguments(dc))
	}
}

func GetEnvironments(dc *topology.DeployConfig) []string {
	envs := []string{}
	if dc.GetKind() == topology.KIND_DINGOFS {

		switch dc.GetRole() {
		case topology.ROLE_FS_MDS,
			topology.ROLE_FS_MDS_CLI:
			if dc.GetCtx().Lookup(topology.CTX_KEY_MDS_VERSION) == topology.CTX_VAL_MDS_V2 {
				envs = configMdsv2ENV(envs, dc)
			}
		case topology.ROLE_COORDINATOR,
			topology.ROLE_STORE:
			envs = configDingoStoreENV(envs, dc)
		case topology.ROLE_DINGODB_EXECUTOR:
			envs = configExecutorENV(envs, dc)
		default:
			// just keep the old envs
			preloads := []string{"/usr/local/lib/libjemalloc.so"}
			if dc.GetEnableRDMA() {
				preloads = append(preloads, "/usr/local/lib/libsmc-preload.so")
			}
			envs = append(envs, fmt.Sprintf("LD_PRELOAD=%s", strings.Join(preloads, " ")))
		}

		env := dc.GetEnv()
		if len(env) > 0 {
			envs = append(envs, strings.Split(env, " ")...)
		}

	} else if dc.GetKind() == topology.KIND_DINGOSTORE {
		switch dc.GetRole() {
		case topology.ROLE_COORDINATOR,
			topology.ROLE_STORE,
			topology.ROLE_DINGODB_DOCUMENT,
			topology.ROLE_DINGODB_INDEX,
			topology.ROLE_DINGODB_DISKANN:
			envs = configDingoStoreENV(envs, dc)
		case topology.ROLE_DINGODB_EXECUTOR,
			topology.ROLE_DINGODB_PROXY,
			topology.ROLE_DINGODB_WEB:
			envs = configExecutorENV(envs, dc)
		default:
			// just keep the old envs
		}
		envs = configDingoStoreENV(envs, dc)
		if len(dc.GetEnv()) > 0 {
			env := strings.Split(dc.GetEnv(), " ")
			envs = append(envs, env...)
		}
	} else if dc.GetKind() == topology.KIND_DINGODB {
		switch dc.GetRole() {
		case topology.ROLE_COORDINATOR,
			topology.ROLE_STORE,
			topology.ROLE_DINGODB_DOCUMENT,
			topology.ROLE_DINGODB_INDEX,
			topology.ROLE_DINGODB_DISKANN:
			envs = configDingoVectorENV(envs, dc)
		case topology.ROLE_DINGODB_EXECUTOR,
			topology.ROLE_DINGODB_PROXY,
			topology.ROLE_DINGODB_WEB:
			envs = configExecutorENV(envs, dc)
		default:
			// just keep the old envs
		}
		env := dc.GetEnv()
		if len(env) > 0 {
			envs = append(envs, strings.Split(env, " ")...)
		}
	}
	return envs
}

func configExecutorENV(envs []string, dc *topology.DeployConfig) []string {
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGODB_EXECUTOR_ROLE, dc.GetRole()))
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGODB_EXECUTOR_HOSTNAME, dc.GetHostname()))
	coordinator_addr := dc.GetDingoStoreCoordinatorAddr()
	if len(coordinator_addr) == 1 {
		coordinator_addr, _ = dc.GetVariables().Get("coordinator_addr")
	}
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGODB_EXECUTOR_COORDINATORS, coordinator_addr))
	return envs
}

func configMdsv2ENV(envs []string, dc *topology.DeployConfig) []string {
	// envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOFS_V2_FLAGS_ROLE, dc.GetRole()))
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOFS_V2_FLAGS_ROLE, topology.ROLE_FS_MDS)) // mds always (change mdsv2 name to mds)
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOFS_V2_FLAGS_CLEAN_LOG, "0"))             // do not clean log
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGO_SERVER_LISTEN_HOST, dc.GetDingoServerListenHost()))
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGO_SERVER_HOST, dc.GetHostname()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGO_SERVER_START_PORT, dc.GetDingoServerPort()))
	coordinator_addr := dc.GetDingoStoreCoordinatorAddr()
	if len(coordinator_addr) == 1 {
		coordinator_addr, _ = dc.GetVariables().Get("coordinator_addr")
	}
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTORE_COORDINATOR_ADDR, coordinator_addr))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOFS_V2_INSTANCE_START_ID, dc.GetDingoInstanceId()))
	return envs
}

func configDingoStoreENV(envs []string, dc *topology.DeployConfig) []string {
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTROE_FLAGS_ROLE, dc.GetRole()))
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTROE_FLAGS_CLEAN_LOG, "1")) // not clean log
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTORE_SERVER_LISTEN_HOST, dc.GetDingoStoreServerListenHost()))
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTORE_RAFT_LISTEN_HOST, dc.GetDingoStoreRaftListenHost()))
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGO_SERVER_HOST, dc.GetHostname()))
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTORE_RAFT_HOST, dc.GetHostname()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_DEFAULT_REPLICA_NUM, dc.GetDingoStoreReplicaNum()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_COORDINATOR_SERVER_START_PORT,
		dc.GetDingoServerPort()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_COORDINATOR_RAFT_START_PORT,
		dc.GetDingoStoreRaftPort()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGO_SERVER_START_PORT, dc.GetDingoServerPort()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_RAFT_START_PORT, dc.GetDingoStoreRaftPort()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_INSTANCE_START_ID, dc.GetDingoInstanceId()))
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTORE_ENABLE_LITE, "false"))
	cluster_coor_srv_peers, err := dc.GetVariables().Get("cluster_coor_srv_peers")
	if err == nil {
		envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTORE_COOR_SRV_PEERS, cluster_coor_srv_peers))
		envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTORE_COORDINATOR_ADDR, cluster_coor_srv_peers))
	}
	cluster_coor_raft_peers, err := dc.GetVariables().Get("cluster_coor_raft_peers")
	if err == nil {
		envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTORE_COOR_RAFT_PEERS, cluster_coor_raft_peers))
	}
	return envs
}

func configDingoVectorENV(envs []string, dc *topology.DeployConfig) []string {
	envs = configDingoStoreENV(envs, dc)
	envs = append(envs, fmt.Sprintf("%s=%s", ENV_DINGOSTORE_DISKANN_SERVER_HOST, dc.GetHostname()))
	str_diskann_server_start_port, err := dc.GetVariables().Get("diskann_server_start_port")
	if err == nil {
		int_diskann_server_start_port, err := strconv.Atoi(str_diskann_server_start_port)
		if err == nil {
			envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_DISKANN_SERVER_START_PORT, int_diskann_server_start_port))
		}
	}
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_DISKANN_INSTANCE_START_ID, dc.GetDingoInstanceId()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_INDEX_SERVER_START_PORT, dc.GetDingoServerPort()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_INDEX_RAFT_START_PORT, dc.GetDingoStoreRaftPort()))
	envs = append(envs, fmt.Sprintf("%s=%d", ENV_DINGOSTORE_INDEX_INSTANCE_START_ID, dc.GetDingoInstanceId()))
	return envs
}

func getUlimits() []string {
	return []string{"nofile=1048576:1048576", "core=-1"}
}

func getMountVolumes(dc *topology.DeployConfig) []step.Volume {
	volumes := []step.Volume{}
	layout := dc.GetProjectLayout() // service container path layout
	logDir := dc.GetLogDir()        // service host log path
	dataDir := dc.GetDataDir()      // service host data path
	sourceCoreDir := dc.GetSourceCoreDir()
	targetCoreDir := dc.GetTargetCoreDir()

	if len(logDir) > 0 {
		volumes = append(volumes, step.Volume{
			HostPath:      logDir,
			ContainerPath: layout.ServiceLogDir,
		})
	}

	if len(dataDir) > 0 && dataDir != comm.SERVICE_DIR_ABSENT {
		volumes = append(volumes, step.Volume{
			HostPath:      dataDir,
			ContainerPath: layout.ServiceDataDir,
		})
	}

	if len(targetCoreDir) > 0 {
		volumes = append(volumes, step.Volume{
			HostPath:      targetCoreDir,
			ContainerPath: sourceCoreDir,
		})
	}

	if dc.GetRole() == topology.ROLE_COORDINATOR ||
		dc.GetRole() == topology.ROLE_STORE ||
		dc.GetRole() == topology.ROLE_DINGODB_DOCUMENT ||
		dc.GetRole() == topology.ROLE_DINGODB_INDEX {
		// mount raft dir
		volumes = append(volumes, step.Volume{
			HostPath:      dc.GetDingoRaftDir(),
			ContainerPath: layout.DingoStoreRaftDir,
		})
		// config store dingodb_br nas volume
		if dc.GetRole() == topology.ROLE_STORE {
			storeConfig := dc.GetServiceConfig()
			if nasVolume, ok := storeConfig["mount.volumes"]; ok {
				volumes = append(volumes, step.Volume{
					HostPath:      nasVolume,
					ContainerPath: nasVolume,
				})
			}
		}
	}
	if dc.GetRole() == topology.ROLE_DINGODB_DOCUMENT {
		// mount document dir
		volumes = append(volumes, step.Volume{
			HostPath:      dc.GetDingoStoreDocDir(),
			ContainerPath: layout.DingoStoreDocumentDir,
		})
	}
	if dc.GetRole() == topology.ROLE_DINGODB_INDEX {
		// mount index vector dir
		volumes = append(volumes, step.Volume{
			HostPath:      dc.GetDingoStoreVectorDir(),
			ContainerPath: layout.DingoStoreVectorDir,
		})
	}

	return volumes
}

func getRestartPolicy(dc *topology.DeployConfig) string {
	switch dc.GetRole() {
	case topology.ROLE_ETCD:
		return POLICY_ALWAYS_RESTART
	case topology.ROLE_COORDINATOR, topology.ROLE_STORE, topology.ROLE_FS_MDS:
		if restartPolicy := dc.GetServiceConfig()["restart_policy"]; restartPolicy != "" {
			return restartPolicy
		}
		return POLICY_ALWAYS_RESTART
	}
	return POLICY_NEVER_RESTART
}

func TrimContainerId(containerId *string) step.LambdaType {
	return func(ctx *context.Context) error {
		items := strings.Split(*containerId, "\n")
		*containerId = items[len(items)-1]
		return nil
	}
}

func NewCreateContainerTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	if dc.GetRole() == topology.ROLE_FS_MDS_CLI {
		return nil, nil
	}
	hc, err := dingoadm.GetHost(dc.GetHost())
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s role=%s", dc.GetHost(), dc.GetRole())
	t := task.NewTask("Create Container", subname, hc.GetSSHConfig())

	// add step to task
	var oldContainerId, containerId string
	clusterId := dingoadm.ClusterId()
	dcId := dc.GetId()
	serviceId := dingoadm.GetServiceId(dcId)
	kind := dc.GetKind()
	role := dc.GetRole()
	hostname := fmt.Sprintf("%s-%s-%s", kind, role, serviceId)
	options := dingoadm.ExecOptions()
	options.ExecWithSudo = false

	t.AddStep(&Step2GetService{ // if service exist, break task
		ServiceId:   serviceId,
		ContainerId: &oldContainerId,
		Storage:     dingoadm.Storage(),
	})

	createDir := []string{dc.GetLogDir(), dc.GetDataDir()}
	if dc.GetRole() == topology.ROLE_COORDINATOR ||
		dc.GetRole() == topology.ROLE_STORE ||
		dc.GetRole() == topology.ROLE_DINGODB_DOCUMENT ||
		dc.GetRole() == topology.ROLE_DINGODB_INDEX {
		createDir = append(createDir, dc.GetDingoRaftDir())
	}
	if dc.GetRole() == topology.ROLE_DINGODB_DOCUMENT {
		createDir = append(createDir, dc.GetDingoStoreDocDir())
	}
	if dc.GetRole() == topology.ROLE_DINGODB_INDEX {
		createDir = append(createDir, dc.GetDingoStoreVectorDir())
	}
	// TODO createDir for dingodb executor /opt/dingo/localStore

	t.AddStep(&step.CreateDirectory{
		Paths:       createDir,
		ExecOptions: options,
	})
	t.AddStep(&step.CreateContainer{
		Image:      dc.GetContainerImage(),
		Command:    getContainerCMD(dc),
		AddHost:    []string{fmt.Sprintf("%s:127.0.0.1", hostname)},
		Envs:       GetEnvironments(dc),
		Hostname:   hostname,
		Init:       true,
		Name:       hostname,
		Privileged: true,
		Restart:    getRestartPolicy(dc), // POLICY_ALWAYS_RESTART
		//--ulimit core=-1: Sets the core dump file size limit to -1, meaning there’s no restriction on the core dump size.
		//--ulimit nofile=65535:65535: Sets both the soft and hard limits for the number of open files to 65535.
		Ulimits:     getUlimits(),
		Volumes:     getMountVolumes(dc),
		Out:         &containerId,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: TrimContainerId(&containerId),
	})
	t.AddStep(&Step2InsertService{
		ClusterId:      clusterId,
		ServiceId:      serviceId,
		ContainerId:    &containerId,
		OldContainerId: &oldContainerId,
		Storage:        dingoadm.Storage(),
	})

	return t, nil
}

func NewCreateMdsv2CliContainerTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	hc, err := dingoadm.GetHost(dc.GetHost())
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s role=%s", dc.GetHost(), dc.GetRole())
	t := task.NewTask("Create mds client Container", subname, hc.GetSSHConfig())

	// add step to task
	var oldContainerId, containerId string
	clusterId := dingoadm.ClusterId()
	dcId := dc.GetId()
	serviceId := dingoadm.GetServiceId(dcId)
	kind := dc.GetKind()
	role := dc.GetRole()
	hostname := fmt.Sprintf("%s-%s-%s", kind, role, serviceId)
	options := dingoadm.ExecOptions()
	options.ExecWithSudo = false

	t.AddStep(&Step2GetService{ // if service exist, break task
		ServiceId:   serviceId,
		ContainerId: &oldContainerId,
		Storage:     dingoadm.Storage(),
	})

	t.AddStep(&step.CreateContainer{
		Image:      dc.GetContainerImage(),
		Entrypoint: "bash",
		Command:    "-c \"while true; do sleep 3600; done\"",
		AddHost:    []string{fmt.Sprintf("%s:127.0.0.1", hostname)},
		Envs:       GetEnvironments(dc),
		Hostname:   hostname,
		Init:       true,
		Name:       hostname,
		Privileged: true,
		Restart:    POLICY_NEVER_RESTART,
		//--ulimit core=-1: Sets the core dump file size limit to -1, meaning there’s no restriction on the core dump size.
		//--ulimit nofile=65535:65535: Sets both the soft and hard limits for the number of open files to 65535.
		Out:         &containerId,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: TrimContainerId(&containerId),
	})
	t.AddStep(&Step2InsertService{
		ClusterId:      clusterId,
		ServiceId:      serviceId,
		ContainerId:    &containerId,
		OldContainerId: &oldContainerId,
		Storage:        dingoadm.Storage(),
	})

	return t, nil
}
