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
 * Created Date: 2022-07-27
 * Author: Jingli Chen (Wine93)
 */
/*
* Project: dingoadm
* Created Date: 2025-03-10
* Author: dongwei (jackblack369)
 */

// __SIGN_BY_WINE93__

package playbook

import (
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/internal/task/task/bs"
	"github.com/dingodb/dingoadm/internal/task/task/checker"
	comm "github.com/dingodb/dingoadm/internal/task/task/common"
	"github.com/dingodb/dingoadm/internal/task/task/fs"
	"github.com/dingodb/dingoadm/internal/task/task/gateway"
	"github.com/dingodb/dingoadm/internal/task/task/monitor"
	pg "github.com/dingodb/dingoadm/internal/task/task/playground"
	"github.com/dingodb/dingoadm/internal/tasks"
)

const (
	// checker
	CHECK_TOPOLOGY int = iota
	CHECK_SSH_CONNECT
	CHECK_PERMISSION
	CHECK_KERNEL_VERSION
	CHECK_KERNEL_MODULE
	CHECK_PORT_IN_USE
	CHECK_DESTINATION_REACHABLE
	START_HTTP_SERVER
	CHECK_NETWORK_FIREWALL
	GET_HOST_DATE
	CHECK_HOST_DATE
	CHECK_CHUNKFILE_POOL
	CHECK_S3
	CLEAN_PRECHECK_ENVIRONMENT

	// common
	PULL_IMAGE
	CREATE_CONTAINER
	CREATE_MDSV2_CLI_CONTAINER
	SYNC_CONFIG
	START_SERVICE
	START_ETCD
	ENABLE_ETCD_AUTH
	START_MDS
	START_CHUNKSERVER
	START_SNAPSHOTCLONE
	START_METASERVER
	START_MDS_V2
	START_COORDINATOR
	START_STORE
	START_TMP_CONTAINER
	STOP_SERVICE
	RESTART_SERVICE
	CREATE_PHYSICAL_POOL
	CREATE_LOGICAL_POOL
	CREATE_META_TABLES
	UPDATE_TOPOLOGY
	INIT_SERVIE_STATUS
	GET_SERVICE_STATUS
	CLEAN_SERVICE
	INIT_SUPPORT
	COLLECT_REPORT
	COLLECT_CURVEADM
	COLLECT_SERVICE
	COLLECT_CLIENT
	BACKUP_ETCD_DATA
	CHECK_MDS_ADDRESS
	CHECK_STORE_HEALTH
	INIT_CLIENT_STATUS
	GET_CLIENT_STATUS
	INSTALL_CLIENT
	UNINSTALL_CLIENT

	// bs
	FORMAT_CHUNKFILE_POOL
	GET_FORMAT_STATUS
	STOP_FORMAT
	BALANCE_LEADER
	START_NEBD_SERVICE
	CREATE_VOLUME
	MAP_IMAGE
	UNMAP_IMAGE

	// monitor
	PULL_MONITOR_IMAGE
	CREATE_MONITOR_CONTAINER
	SYNC_MONITOR_CONFIG
	CLEAN_CONFIG_CONTAINER
	START_MONITOR_SERVICE
	RESTART_MONITOR_SERVICE
	STOP_MONITOR_SERVICE
	INIT_MONITOR_STATUS
	GET_MONITOR_STATUS
	CLEAN_MONITOR

	// bs/target
	START_TARGET_DAEMON
	STOP_TARGET_DAEMON
	ADD_TARGET
	DELETE_TARGET
	LIST_TARGETS

	// fs
	CHECK_CLIENT_S3
	CREATE_DINGOFS
	MOUNT_FILESYSTEM
	UMOUNT_FILESYSTEM

	// polarfs
	DETECT_OS_RELEASE
	INSTALL_POLARFS
	UNINSTALL_POLARFS

	// playground
	CREATE_PLAYGROUND
	INIT_PLAYGROUND
	START_PLAYGROUND
	REMOVE_PLAYGROUND
	GET_PLAYGROUND_STATUS

	// gateway
	START_GATEWAY

	// unknown
	UNKNOWN
)

func (p *Playbook) createTasks(step *PlaybookStep) (*tasks.Tasks, error) {
	// (1) default tasks execute options
	config, err := NewSmartConfig(step.Configs)
	if err != nil {
		return nil, err
	}

	// (2) set key-value pair for options
	for k, v := range step.Options {
		p.dingoadm.MemStorage().Set(k, v)
	}

	// (3) create task one by one and added into tasks
	var t *task.Task
	once := map[string]bool{}
	dingoadm := p.dingoadm
	ts := tasks.NewTasks()
	for i := 0; i < config.Len(); i++ {
		// only need to execute task once per host
		switch step.Type {
		case CHECK_SSH_CONNECT,
			GET_HOST_DATE:
			host := config.GetDC(i).GetHost()
			if once[host] {
				continue
			}
			once[host] = true
		case PULL_IMAGE:
			host := config.GetDC(i).GetHost()
			image := config.GetDC(i).GetContainerImage()
			if once[host+"_"+image] {
				continue
			}
			once[host+"_"+image] = true
		}

		switch step.Type {
		// checker
		case CHECK_TOPOLOGY:
			t, err = checker.NewCheckTopologyTask(dingoadm, nil)
		case CHECK_SSH_CONNECT:
			t, err = checker.NewCheckSSHConnectTask(dingoadm, config.GetDC(i))
		case CHECK_PERMISSION:
			t, err = checker.NewCheckPermissionTask(dingoadm, config.GetDC(i))
		case CHECK_KERNEL_VERSION:
			t, err = checker.NewCheckKernelVersionTask(dingoadm, config.GetDC(i))
		case CHECK_KERNEL_MODULE:
			t, err = checker.NewCheckKernelModuleTask(dingoadm, config.GetCC(i))
		case CHECK_PORT_IN_USE:
			t, err = checker.NewCheckPortInUseTask(dingoadm, config.GetDC(i))
		case CHECK_DESTINATION_REACHABLE:
			t, err = checker.NewCheckDestinationReachableTask(dingoadm, config.GetDC(i))
		case START_HTTP_SERVER:
			t, err = checker.NewStartHTTPServerTask(dingoadm, config.GetDC(i))
		case CHECK_NETWORK_FIREWALL:
			t, err = checker.NewCheckNetworkFirewallTask(dingoadm, config.GetDC(i))
		case GET_HOST_DATE:
			t, err = checker.NewGetHostDate(dingoadm, config.GetDC(i))
		case CHECK_HOST_DATE:
			t, err = checker.NewCheckDate(dingoadm, nil)
		case CHECK_CHUNKFILE_POOL:
			t, err = checker.NewCheckChunkfilePoolTask(dingoadm, config.GetDC(i))
		case CHECK_S3:
			t, err = checker.NewCheckS3Task(dingoadm, config.GetDC(i))
		case CHECK_MDS_ADDRESS:
			t, err = checker.NewCheckMdsAddressTask(dingoadm, config.GetCC(i))
		case CHECK_STORE_HEALTH:
			t, err = comm.NewCheckStoreHealthTask(dingoadm, config.GetDC(i))
		case CLEAN_PRECHECK_ENVIRONMENT:
			t, err = checker.NewCleanEnvironmentTask(dingoadm, config.GetDC(i))
		// common
		case PULL_IMAGE:
			t, err = comm.NewPullImageTask(dingoadm, config.GetDC(i))
		case CREATE_CONTAINER:
			t, err = comm.NewCreateContainerTask(dingoadm, config.GetDC(i))
		case CREATE_MDSV2_CLI_CONTAINER:
			t, err = comm.NewCreateMdsv2CliContainerTask(dingoadm, config.GetDC(i))
		case SYNC_CONFIG:
			t, err = comm.NewSyncConfigTask(dingoadm, config.GetDC(i))
		case START_SERVICE,
			START_ETCD,
			START_MDS,
			START_CHUNKSERVER,
			START_SNAPSHOTCLONE,
			START_METASERVER,
			START_MDS_V2,
			START_COORDINATOR,
			START_STORE,
			START_TMP_CONTAINER:
			t, err = comm.NewStartServiceTask(dingoadm, config.GetDC(i))
		case ENABLE_ETCD_AUTH:
			t, err = comm.NewEnableEtcdAuthTask(dingoadm, config.GetDC(i))
		case STOP_SERVICE:
			t, err = comm.NewStopServiceTask(dingoadm, config.GetDC(i))
		case RESTART_SERVICE:
			t, err = comm.NewRestartServiceTask(dingoadm, config.GetDC(i))
		case CREATE_PHYSICAL_POOL,
			CREATE_LOGICAL_POOL:
			t, err = comm.NewCreateTopologyTask(dingoadm, config.GetDC(i))
		case CREATE_META_TABLES:
			t, err = comm.NewCreateMetaTablesTask(dingoadm, config.GetDC(i))
		case UPDATE_TOPOLOGY:
			t, err = comm.NewUpdateTopologyTask(dingoadm, nil)
		case INIT_SERVIE_STATUS:
			t, err = comm.NewInitServiceStatusTask(dingoadm, config.GetDC(i))
		case GET_SERVICE_STATUS:
			t, err = comm.NewGetServiceStatusTask(dingoadm, config.GetDC(i))
		case CLEAN_SERVICE:
			t, err = comm.NewCleanServiceTask(dingoadm, config.GetDC(i))
		case INIT_SUPPORT:
			t, err = comm.NewInitSupportTask(dingoadm, config.GetDC(i))
		case COLLECT_REPORT:
			t, err = comm.NewCollectReportTask(dingoadm, config.GetDC(i))
		case COLLECT_CURVEADM:
			t, err = comm.NewCollectCurveAdmTask(dingoadm, config.GetDC(i))
		case COLLECT_SERVICE:
			t, err = comm.NewCollectServiceTask(dingoadm, config.GetDC(i))
		case COLLECT_CLIENT:
			t, err = comm.NewCollectClientTask(dingoadm, config.GetAny(i))
		case BACKUP_ETCD_DATA:
			t, err = comm.NewBackupEtcdDataTask(dingoadm, config.GetDC(i))
		case INIT_CLIENT_STATUS:
			t, err = comm.NewInitClientStatusTask(dingoadm, config.GetAny(i))
		case GET_CLIENT_STATUS:
			t, err = comm.NewGetClientStatusTask(dingoadm, config.GetAny(i))
		case INSTALL_CLIENT:
			t, err = comm.NewInstallClientTask(dingoadm, config.GetCC(i))
		case UNINSTALL_CLIENT:
			t, err = comm.NewUninstallClientTask(dingoadm, nil)
		// bs
		case FORMAT_CHUNKFILE_POOL:
			t, err = bs.NewFormatChunkfilePoolTask(dingoadm, config.GetFC(i))
		case GET_FORMAT_STATUS:
			t, err = bs.NewGetFormatStatusTask(dingoadm, config.GetFC(i))
		case STOP_FORMAT:
			t, err = bs.NewStopFormatTask(dingoadm, config.GetFC(i))
		case BALANCE_LEADER:
			t, err = bs.NewBalanceTask(dingoadm, config.GetDC(i))
		case START_NEBD_SERVICE:
			t, err = bs.NewStartNEBDServiceTask(dingoadm, config.GetCC(i))
		case CREATE_VOLUME:
			t, err = bs.NewCreateVolumeTask(dingoadm, config.GetCC(i))
		case MAP_IMAGE:
			t, err = bs.NewMapTask(dingoadm, config.GetCC(i))
		case UNMAP_IMAGE:
			t, err = bs.NewUnmapTask(dingoadm, nil)
		// bs/target
		case START_TARGET_DAEMON:
			t, err = bs.NewStartTargetDaemonTask(dingoadm, config.GetCC(i))
		case STOP_TARGET_DAEMON:
			t, err = bs.NewStopTargetDaemonTask(dingoadm, nil)
		case ADD_TARGET:
			t, err = bs.NewAddTargetTask(dingoadm, config.GetCC(i))
		case DELETE_TARGET:
			t, err = bs.NewDeleteTargetTask(dingoadm, nil)
		case LIST_TARGETS:
			t, err = bs.NewListTargetsTask(dingoadm, nil)
		// fs
		case CHECK_CLIENT_S3:
			t, err = checker.NewClientS3ConfigureTask(dingoadm, config.GetCC(i))
		case CREATE_DINGOFS:
			t, err = fs.NewCreateDingoFSTask(dingoadm, config.GetCC(i))
		case MOUNT_FILESYSTEM:
			t, err = fs.NewMountFSTask(dingoadm, config.GetCC(i))
		case UMOUNT_FILESYSTEM:
			t, err = fs.NewUmountFSTask(dingoadm, config.GetCC(i))
		// polarfs
		case DETECT_OS_RELEASE:
			t, err = bs.NewDetectOSReleaseTask(dingoadm, nil)
		case INSTALL_POLARFS:
			t, err = bs.NewInstallPolarFSTask(dingoadm, config.GetCC(i))
		case UNINSTALL_POLARFS:
			t, err = bs.NewUninstallPolarFSTask(dingoadm, nil)
		// playground
		case CREATE_PLAYGROUND:
			t, err = pg.NewCreatePlaygroundTask(dingoadm, config.GetPGC(i))
		case INIT_PLAYGROUND:
			t, err = pg.NewInitPlaygroundTask(dingoadm, config.GetPGC(i))
		case START_PLAYGROUND:
			t, err = pg.NewStartPlaygroundTask(dingoadm, config.GetPGC(i))
		case REMOVE_PLAYGROUND:
			t, err = pg.NewRemovePlaygroundTask(dingoadm, config.GetAny(i))
		case GET_PLAYGROUND_STATUS:
			t, err = pg.NewGetPlaygroundStatusTask(dingoadm, config.GetAny(i))
		// monitor
		case PULL_MONITOR_IMAGE:
			t, err = monitor.NewPullImageTask(dingoadm, config.GetMC(i))
		case CREATE_MONITOR_CONTAINER:
			t, err = monitor.NewCreateContainerTask(dingoadm, config.GetMC(i))
		case SYNC_MONITOR_CONFIG:
			t, err = monitor.NewSyncConfigTask(dingoadm, config.GetMC(i))
		case CLEAN_CONFIG_CONTAINER:
			t, err = monitor.NewCleanConfigContainerTask(dingoadm, config.GetMC(i))
		case START_MONITOR_SERVICE:
			t, err = monitor.NewStartServiceTask(dingoadm, config.GetMC(i))
		case RESTART_MONITOR_SERVICE:
			t, err = monitor.NewRestartServiceTask(dingoadm, config.GetMC(i))
		case STOP_MONITOR_SERVICE:
			t, err = monitor.NewStopServiceTask(dingoadm, config.GetMC(i))
		case INIT_MONITOR_STATUS:
			t, err = monitor.NewInitMonitorStatusTask(dingoadm, config.GetMC(i))
		case GET_MONITOR_STATUS:
			t, err = monitor.NewGetMonitorStatusTask(dingoadm, config.GetMC(i))
		case CLEAN_MONITOR:
			t, err = monitor.NewCleanMonitorTask(dingoadm, config.GetMC(i))
		case START_GATEWAY:
			t, err = gateway.NewStartGatewayTask(dingoadm, config.GetGC())

		default:
			return nil, errno.ERR_UNKNOWN_TASK_TYPE.
				F("task type: %d", step.Type)
		}

		if err != nil {
			return nil, err // already is error code
		} else if t == nil {
			continue
		}

		if config.GetType() == TYPE_CONFIG_DEPLOY { // merge task status into one
			t.SetTid(config.GetDC(i).GetId())
			t.SetPtid(config.GetDC(i).GetParentId())
		}
		ts.AddTask(t)
	}

	return ts, nil
}
