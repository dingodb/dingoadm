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
 * Created Date: 2022-05-20
 * Author: Jingli Chen (Wine93)
 */

// __SIGN_BY_WINE93__

package common

import (
	"github.com/dingodb/dingoadm/internal/configure/topology"
)

var (
	ROLES = []string{
		topology.ROLE_ETCD,
		topology.ROLE_MDS,
		topology.ROLE_CHUNKSERVER,
		topology.ROLE_SNAPSHOTCLONE,
		topology.ROLE_METASERVER,
	}
)

// task options
const (
	// cluster
	KEY_ENV_ACTIVATE_CLUSTER = "DINGO_CLUSTER"

	// common
	KEY_ALL_DEPLOY_CONFIGS    = "ALL_DEPLOY_CONFIGS"
	KEY_POOLSET               = "KEY_POOLSET"
	KEY_CREATE_POOL_TYPE      = "POOL_TYPE"
	POOL_TYPE_LOGICAL         = "logicalpool"
	POOL_TYPE_PHYSICAL        = "physicalpool"
	POOLSET                   = "poolset"
	POOLSET_DISK_TYPE         = "poolset-disktype"
	KEY_NUMBER_OF_CHUNKSERVER = "NUMBER_OF_CHUNKSERVER"

	// format
	KEY_ALL_FORMAT_STATUS = "ALL_FORMAT_STATUS"

	// check
	KEY_CHECK_WITH_WEAK          = "CHECK_WITH_WEAK"
	KEY_CHECK_KERNEL_MODULE_NAME = "CHECK_KERNEL_MODULE_NAME"
	KEY_CHECK_SKIP_SNAPSHOECLONE = "CHECK_SKIP_SNAPSHOTCLONE"
	KEY_ALL_HOST_DATE            = "ALL_HOST_DATE"
	KEY_ALLOW_ABSENT             = "ALLOW_ABSENT"

	// scale-out / migrate
	KEY_SCALE_OUT_CLUSTER = "SCALE_OUT_CLUSTER"
	KEY_MIGRATE_SERVERS   = "MIGRATE_SERVERS"
	KEY_NEW_TOPOLOGY_DATA = "NEW_TOPOLOGY_DATA"

	// status
	KEY_ALL_SERVICE_STATUS = "ALL_SERVICE_STATUS"
	SERVICE_STATUS_CLEANED = "Cleaned"
	SERVICE_STATUS_LOSED   = "Losed"
	SERVICE_STATUS_UNKNOWN = "Unknown"

	// clean
	KEY_CLEAN_ITEMS      = "CLEAN_ITEMS"
	KEY_CLEAN_BY_RECYCLE = "CLEAN_BY_RECYCLE"
	CLEAN_ITEM_LOG       = "log"
	CLEAN_ITEM_DATA      = "data"
	CLEAN_ITEM_CONTAINER = "container"
	CLEAN_ITEM_RAFT      = "raft"
	CLEANED_CONTAINER_ID = "-"

	// client
	KEY_CLIENT_HOST           = "CLIENT_HOST"
	KEY_CLIENT_KIND           = "CLIENT_KIND"
	KEY_ALL_CLIENT_STATUS     = "ALL_CLIENT_STATUS"
	KEY_CLIENT_STATUS_VERBOSE = "CLIENT_STATUS_VERBOSE"
	KEY_MAP_OPTIONS           = "MAP_OPTIONS"
	KEY_MOUNT_OPTIONS         = "MOUNT_OPTIONS"
	KEY_USE_LOCAL_IMAGE       = "USE_LOCAL_IMAGE"
	KEY_USE_NEW_DINGO         = "USE_NEW_DINGO"
	CLIENT_STATUS_LOSED       = "Losed"
	CLIENT_STATUS_UNKNOWN     = "Unknown"
	KERNERL_MODULE_NBD        = "nbd"
	KERNERL_MODULE_FUSE       = "fuse"

	// polarfs
	KEY_POLARFS_HOST   = "POLARFS_HOST"
	KEY_OS_RELEASE     = "OS_RELEASE"
	OS_RELEASE_DEBIAN  = "debian"
	OS_RELEASE_UBUNTU  = "ubuntu"
	OS_RELEASE_CENTOS  = "centos"
	OS_RELEASE_UNKNOWN = "unknown"

	// collect
	KEY_SUPPORT_UPLOAD_URL_FORMAT = "SUPPORT_UPLOAD_URL"
	KEY_SECRET                    = "SECRET"
	KEY_ALL_CLIENT_IDS            = "ALL_CLIENT_IDS"

	// target
	KEY_TARGET_OPTIONS = "TARGET_OPTIONS"
	KEY_ALL_TARGETS    = "ALL_TARGETS"

	// playground
	KEY_ALL_PLAYGROUNDS_STATUS = "ALL_PLAYGROUNDS_STATUS"
	PLAYGROUDN_STATUS_LOSED    = "Losed"

	// monitor
	KEY_MONITOR_HOST     = "MONITOR_HOST"
	KEY_SERVICE_HOSTS    = "SERVICE_HOSTS"
	KEY_MONITOR_STATUS   = "MONITOR_STATUS"
	CLEANED_MONITOR_CONF = "-"

	// gateway
	GATEWAY_NAME         = "GATEWAY_NAME"
	GATEWAY_HOST         = "GATEWAY_HOST"
	GATEWAY_LISTEN_ADDR  = "GATEWAY_LISTEN_ADDR"
	GATEWAY_CONSOLE_ADDR = "GATEWAY_CONSOLE_ADDR"
	GATEWAY_MOUNTPOINT   = "GATEWAY_MOUNTPOINT"
	MDSADDR              = "mdsaddr"

	// delimiter symbol
	CLIENT_CONFIG_DELIMITER   = "="
	TOOLS_V2_CONFIG_DELIMITER = ": "

	// mdsv2
	KEY_SKIP_TMP = "SKIP_TMP"
)

// others
const (
	AUDIT_STATUS_ABORT = iota
	AUDIT_STATUS_SUCCESS
	AUDIT_STATUS_FAIL
	AUDIT_STATUS_CANCEL
)
