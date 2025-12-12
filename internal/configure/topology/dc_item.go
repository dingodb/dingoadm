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
 * Created Date: 2021-12-24
 * Author: Jingli Chen (Wine93)
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

// __SIGN_BY_WINE93__

package topology

import (
	"path"
	"strings"
)

const (
	REQUIRE_ANY = iota
	REQUIRE_INT
	REQUIRE_STRING
	REQUIRE_BOOL
	REQUIRE_POSITIVE_INTEGER
	REQUIRE_MAP

	// default value
	DEFAULT_REPORT_USAGE                    = false
	DEFAULT_CURVEBS_CONTAINER_IMAGE         = "opencurvedocker/curvebs:latest"
	DEFAULT_DINGOFS_CONTAINER_IMAGE         = "dingodatabase/dingofs:latest"
	DEFAULT_ETCD_LISTEN_PEER_PORT           = 2380
	DEFAULT_ETCD_LISTEN_CLIENT_PORT         = 2379
	DEFAULT_MDS_LISTEN_PORT                 = 6700
	DEFAULT_MDS_LISTEN_DUMMY_PORT           = 7700
	DEFAULT_CHUNKSERVER_LISTN_PORT          = 8200
	DEFAULT_SNAPSHOTCLONE_LISTEN_PORT       = 5555
	DEFAULT_COORDINATOR_SERVER_PORT         = 6500
	DEFAULT_COORDINATOR_RAFT_PORT           = 7500
	DEFAULT_STORE_SERVER_PORT               = 6600
	DEFAULT_STORE_RAFT_PORT                 = 7600
	DEFAULT_STORE_SERVER_LISTEN_HOST        = "0.0.0.0"
	DEFAULT_STORE_RAFT_LISTEN_HOST          = "0.0.0.0"
	DEFAULT_STORE_REPLICA_NUM               = 3
	DEFAULT_STORE_INSTANCE_START_ID         = 1001
	DEFAULT_SNAPSHOTCLONE_LISTEN_DUMMY_PORT = 8081
	DEFAULT_SNAPSHOTCLONE_LISTEN_PROXY_PORT = 8080
	DEFAULT_METASERVER_LISTN_PORT           = 6800
	DEFAULT_METASERVER_LISTN_EXTARNAL_PORT  = 7800
	DEFAULT_DINGO_SERVER_LISTEN_HOST        = "0.0.0.0"
	DEFAULT_FS_MDS_LISTEN_PORT              = 6900
	DEFAULT_ENABLE_EXTERNAL_SERVER          = false
	DEFAULT_CHUNKSERVER_COPYSETS            = 100 // copysets per chunkserver
	DEFAULT_METASERVER_COPYSETS             = 100 // copysets per metaserver
	DEFAULT_DINGODB_EXECUTOR_SERVER_PORT    = 8765
	DEFAULT_DINGODB_EXECUTOR_MYSQL_PORT     = 3307
	DEFAULT_DOCUMENT_SERVER_PORT            = 23001
	DEFAULT_DOCUMENT_RAFT_PORT              = 23101
	DEFAULT_INDEX_SERVER_PORT               = 21001
	DEFAULT_INDEX_RAFT_PORT                 = 21101
	DEFAULT_DISKANN_SERVER_PORT             = 24001
	DEFAULT_DINGODB_WEB_SERVER_PORT         = 13001
	DEFAULT_DINGODB_PROXY_SERVER_PORT       = 13000
	DEFAULT_DINGODB_WEB_EXPORT_PORT         = 19100
)

type (
	// config item
	item struct {
		kind         string
		key          string
		require      int
		exclude      bool        // exclude for service config
		defaultValue interface{} // nil means no default value
	}

	itemSet struct {
		items    []*item
		key2item map[string]*item
	}
)

// you should add config item to itemset iff you want to:
//
//	(1) check the configuration item value, like type, valid value OR
//	(2) filter out the configuration item for service config OR
//	(3) set the default value for configuration item
var (

	// itemset is config service var on host
	itemset = &itemSet{
		items:    []*item{},
		key2item: map[string]*item{},
	}

	CONFIG_PREFIX = itemset.insert(
		KIND_DINGO,
		"prefix",
		REQUIRE_STRING,
		true,
		func(dc *DeployConfig) interface{} {
			root_dir := LAYOUT_DINGO_ROOT_DIR
			switch dc.GetKind() {
			case KIND_DINGOFS:
				if dc.GetRole() == ROLE_FS_MDS {
					if dc.GetCtx().Lookup(CTX_KEY_MDS_VERSION) == CTX_VAL_MDS_V2 {
						root_dir = path.Join(LAYOUT_DINGOFS_ROOT_DIR, "dist", ROLE_FS_MDS) // change root dir from mdsv2 (dc.GetRole()) to mds
					} else {
						root_dir = path.Join(LAYOUT_DINGOFS_ROOT_DIR, dc.GetRole())
					}
				} else if dc.GetRole() == ROLE_COORDINATOR || dc.GetRole() == ROLE_STORE {
					root_dir = LAYOUT_DINGOSTORE_ROOT_DIR
				} else if dc.GetRole() == ROLE_DINGODB_EXECUTOR {
					root_dir = LAYOUT_DINGDB_DINGO_ROOT_DIR
				} else {
					root_dir = path.Join(LAYOUT_DINGOFS_ROOT_DIR, dc.GetRole())
				}
			case KIND_DINGOSTORE:
				if dc.GetRole() == ROLE_COORDINATOR || dc.GetRole() == ROLE_STORE {
					root_dir = LAYOUT_DINGOSTORE_ROOT_DIR
				} else if dc.GetRole() == ROLE_DINGODB_EXECUTOR {
					root_dir = LAYOUT_DINGDB_DINGO_ROOT_DIR
				} else {
					root_dir = LAYOUT_DINGOSTORE_DIST_DIR
				}
			case KIND_DINGODB:
				dc_role := dc.GetRole()
				switch dc_role {
				case ROLE_COORDINATOR, ROLE_STORE, ROLE_DINGODB_DOCUMENT, ROLE_DINGODB_INDEX, ROLE_DINGODB_DISKANN:
					root_dir = LAYOUT_DINGOSTORE_ROOT_DIR
				case ROLE_DINGODB_PROXY, ROLE_DINGODB_WEB, ROLE_DINGODB_EXECUTOR:
					root_dir = LAYOUT_DINGDB_DINGO_ROOT_DIR
				}
			default:
				root_dir = path.Join(LAYOUT_DINGO_ROOT_DIR, dc.GetRole())
			}
			return root_dir
		},
	)

	CONFIG_REPORT_USAGE = itemset.insert(
		KIND_DINGOFS,
		"report_usage",
		REQUIRE_BOOL,
		true,
		DEFAULT_REPORT_USAGE,
	)

	CONFIG_CONTAINER_IMAGE = itemset.insert(
		KIND_DINGO,
		"container_image",
		REQUIRE_STRING,
		true,
		func(dc *DeployConfig) interface{} {
			if dc.GetKind() == KIND_CURVEBS {
				return DEFAULT_CURVEBS_CONTAINER_IMAGE
			}
			return DEFAULT_DINGOFS_CONTAINER_IMAGE
		},
	)

	CONFIG_LOG_DIR = itemset.insert(
		KIND_DINGO,
		"log_dir",
		REQUIRE_STRING,
		true,
		nil,
	)

	CONFIG_DATA_DIR = itemset.insert(
		KIND_DINGO,
		"data_dir",
		REQUIRE_STRING,
		true,
		nil,
	)

	CONFIG_SEQ_OFFSET = itemset.insert(
		KIND_DINGO,
		"sequence_offset",
		REQUIRE_POSITIVE_INTEGER,
		true,
		nil,
	)

	CONFIG_SOURCE_CORE_DIR = itemset.insert(
		KIND_DINGO,
		"source_core_dir",
		REQUIRE_STRING,
		true,
		nil,
	)

	CONFIG_TARGET_CORE_DIR = itemset.insert(
		KIND_DINGO,
		"target_core_dir",
		REQUIRE_STRING,
		true,
		nil,
	)

	CONFIG_ENV = itemset.insert(
		KIND_DINGO,
		"env",
		REQUIRE_STRING,
		true,
		nil,
	)

	CONFIG_LISTEN_IP = itemset.insert(
		KIND_DINGO,
		"listen.ip",
		REQUIRE_STRING,
		true,
		func(dc *DeployConfig) interface{} {
			return dc.GetHostname()
		},
	)

	CONFIG_LISTEN_PORT = itemset.insert(
		KIND_DINGOFS,
		"listen.port",
		REQUIRE_POSITIVE_INTEGER,
		true,
		func(dc *DeployConfig) interface{} {
			switch dc.GetRole() {
			case ROLE_ETCD:
				return DEFAULT_ETCD_LISTEN_PEER_PORT
			// case ROLE_MDS_V1:
			// 	return DEFAULT_MDS_LISTEN_PORT
			case ROLE_CHUNKSERVER:
				return DEFAULT_CHUNKSERVER_LISTN_PORT
			case ROLE_SNAPSHOTCLONE:
				return DEFAULT_SNAPSHOTCLONE_LISTEN_PORT
			case ROLE_METASERVER:
				return DEFAULT_METASERVER_LISTN_PORT
			case ROLE_FS_MDS,
				ROLE_FS_MDS_CLI:
				return DEFAULT_FS_MDS_LISTEN_PORT
			case ROLE_COORDINATOR:
				return DEFAULT_COORDINATOR_SERVER_PORT
			case ROLE_STORE:
				return DEFAULT_STORE_SERVER_PORT
			case ROLE_DINGODB_EXECUTOR:
				return DEFAULT_DINGODB_EXECUTOR_MYSQL_PORT
			}
			return nil
		},
	)

	CONFIG_LISTEN_CLIENT_PORT = itemset.insert(
		KIND_DINGOFS,
		"listen.client_port",
		REQUIRE_POSITIVE_INTEGER,
		true,
		DEFAULT_ETCD_LISTEN_CLIENT_PORT,
	)

	CONFIG_LISTEN_DUMMY_PORT = itemset.insert(
		KIND_DINGOFS,
		"listen.dummy_port",
		REQUIRE_POSITIVE_INTEGER,
		true,
		func(dc *DeployConfig) interface{} {
			switch dc.GetRole() {
			case ROLE_FS_MDS:
				return DEFAULT_MDS_LISTEN_DUMMY_PORT
			case ROLE_SNAPSHOTCLONE:
				return DEFAULT_SNAPSHOTCLONE_LISTEN_DUMMY_PORT
			}
			return nil
		},
	)

	CONFIG_LISTEN_PROXY_PORT = itemset.insert(
		KIND_DINGOFS,
		"listen.proxy_port",
		REQUIRE_POSITIVE_INTEGER,
		true,
		DEFAULT_SNAPSHOTCLONE_LISTEN_PROXY_PORT,
	)

	CONFIG_LISTEN_EXTERNAL_IP = itemset.insert(
		KIND_DINGOFS,
		"listen.external_ip",
		REQUIRE_STRING,
		true,
		func(dc *DeployConfig) interface{} {
			return dc.GetHostname()
		},
	)

	CONFIG_LISTEN_EXTERNAL_PORT = itemset.insert(
		KIND_DINGOFS,
		"listen.external_port",
		REQUIRE_POSITIVE_INTEGER,
		true,
		func(dc *DeployConfig) interface{} {
			if dc.GetRole() == ROLE_METASERVER {
				return DEFAULT_METASERVER_LISTN_EXTARNAL_PORT
			}
			return dc.GetListenPort()
		},
	)

	CONFIG_ENABLE_EXTERNAL_SERVER = itemset.insert(
		KIND_DINGOFS,
		"global.enable_external_server",
		REQUIRE_BOOL,
		false,
		DEFAULT_ENABLE_EXTERNAL_SERVER,
	)

	CONFIG_COPYSETS = itemset.insert(
		KIND_DINGOFS,
		"copysets",
		REQUIRE_POSITIVE_INTEGER,
		true,
		func(dc *DeployConfig) interface{} {
			if dc.GetRole() == ROLE_CHUNKSERVER {
				return DEFAULT_CHUNKSERVER_COPYSETS
			}
			return DEFAULT_METASERVER_COPYSETS
		},
	)

	CONFIG_S3_ACCESS_KEY = itemset.insert(
		KIND_DINGOFS,
		"s3.ak",
		REQUIRE_STRING,
		false,
		nil,
	)

	CONFIG_S3_SECRET_KEY = itemset.insert(
		KIND_DINGOFS,
		"s3.sk",
		REQUIRE_STRING,
		false,
		nil,
	)

	CONFIG_S3_ADDRESS = itemset.insert(
		KIND_DINGOFS,
		"s3.nos_address",
		REQUIRE_STRING,
		false,
		nil,
	)

	CONFIG_S3_BUCKET_NAME = itemset.insert(
		KIND_DINGOFS,
		"s3.snapshot_bucket_name",
		REQUIRE_STRING,
		false,
		nil,
	)

	CONFIG_ENABLE_RDMA = itemset.insert(
		KIND_DINGOFS,
		"enable_rdma",
		REQUIRE_BOOL,
		true,
		false,
	)

	CONFIG_ENABLE_RENAMEAT2 = itemset.insert(
		KIND_DINGOFS,
		"fs.enable_renameat2",
		REQUIRE_BOOL,
		false,
		true,
	)

	CONFIG_ENABLE_CHUNKFILE_POOL = itemset.insert(
		KIND_DINGOFS,
		"chunkfilepool.enable_get_chunk_from_pool",
		REQUIRE_BOOL,
		false,
		true,
	)

	CONFIG_VARIABLE = itemset.insert(
		KIND_DINGO,
		"variable",
		REQUIRE_STRING,
		true,
		nil,
	)

	CONFIG_ETCD_AUTH_ENABLE = itemset.insert(
		KIND_DINGOFS,
		"etcd.auth.enable",
		REQUIRE_BOOL,
		false,
		false,
	)

	CONFIG_ETCD_AUTH_USERNAME = itemset.insert(
		KIND_DINGOFS,
		"etcd.auth.username",
		REQUIRE_STRING,
		false,
		nil,
	)

	CONFIG_ETCD_AUTH_PASSWORD = itemset.insert(
		KIND_DINGOFS,
		"etcd.auth.password",
		REQUIRE_STRING,
		false,
		nil,
	)

	CONFIG_DINGO_STORE_RAFT_DIR = itemset.insert(
		KIND_DINGO,
		"raft_dir",
		REQUIRE_STRING,
		true,
		nil,
	)

	CONFIG_DINGO_STORE_SERVER_PORT = itemset.insert(
		KIND_DINGO,
		"server.port",
		REQUIRE_POSITIVE_INTEGER,
		true,
		func(dc *DeployConfig) interface{} {
			switch dc.GetRole() {
			case ROLE_COORDINATOR:
				return DEFAULT_COORDINATOR_SERVER_PORT
			case ROLE_STORE:
				return DEFAULT_STORE_SERVER_PORT
			case ROLE_FS_MDS:
				return DEFAULT_FS_MDS_LISTEN_PORT
			case ROLE_DINGODB_DOCUMENT:
				return DEFAULT_DOCUMENT_SERVER_PORT
			case ROLE_DINGODB_INDEX:
				return DEFAULT_INDEX_SERVER_PORT
			case ROLE_DINGODB_DISKANN:
				return DEFAULT_DISKANN_SERVER_PORT
			}
			return nil
		},
	)

	CONFIG_DINGO_STORE_RAFT_PORT = itemset.insert(
		KIND_DINGO,
		"raft.port",
		REQUIRE_POSITIVE_INTEGER,
		true,
		func(dc *DeployConfig) interface{} {
			switch dc.GetRole() {
			case ROLE_COORDINATOR:
				return DEFAULT_COORDINATOR_RAFT_PORT
			case ROLE_STORE:
				return DEFAULT_STORE_RAFT_PORT
			case ROLE_DINGODB_DOCUMENT:
				return DEFAULT_DOCUMENT_RAFT_PORT
			case ROLE_DINGODB_INDEX:
				return DEFAULT_INDEX_RAFT_PORT
			}
			return nil
		},
	)

	CONFFIG_DINGO_STORE_SERVER_LISTEN_HOST = itemset.insert(
		KIND_DINGO,
		"server_listen_host",
		REQUIRE_STRING,
		true,
		func(dc *DeployConfig) interface{} {
			return DEFAULT_STORE_SERVER_LISTEN_HOST
		},
	)

	CONFFIG_DINGO_STORE_RAFT_LISTEN_HOST = itemset.insert(
		KIND_DINGO,
		"raft_listen_host",
		REQUIRE_STRING,
		true,
		func(dc *DeployConfig) interface{} {
			return DEFAULT_STORE_RAFT_LISTEN_HOST
		},
	)

	CONFIG_DINGO_STORE_REPLICA_NUM = itemset.insert(
		KIND_DINGO,
		"default_replica_num",
		REQUIRE_POSITIVE_INTEGER,
		true,
		func(dc *DeployConfig) interface{} {
			return DEFAULT_STORE_REPLICA_NUM
		},
	)

	CONFIG_INSTANCE_START_ID = itemset.insert(
		KIND_DINGO,
		"instance_start_id",
		REQUIRE_POSITIVE_INTEGER,
		true,
		func(dc *DeployConfig) interface{} {
			if dc.GetInstances() > 0 {
				return DEFAULT_STORE_INSTANCE_START_ID + dc.GetHostSequence()*dc.GetInstances() + dc.GetInstancesSequence()
			}
			return DEFAULT_STORE_INSTANCE_START_ID + dc.GetHostSequence() + dc.GetInstancesSequence()
		},
	)

	CONFIG_DINGOSTORE_COORDINATOR_ADDR = itemset.insert(
		KIND_DINGO,
		"coordinator_addr",
		REQUIRE_STRING,
		true,
		func(dc *DeployConfig) interface{} {
			value, err := dc.GetVariables().Get("coordinator_addr")
			if err != nil {
				return "-"
			}
			return value
		},
	)

	CONFFIG_DINGO_SERVER_LISTEN_HOST = itemset.insert(
		KIND_DINGO,
		"server_listen_host",
		REQUIRE_STRING,
		true,
		func(dc *DeployConfig) interface{} {
			return DEFAULT_DINGO_SERVER_LISTEN_HOST
		},
	)

	CONFIG_DINGO_STORE_DOCUMENT_DIR = itemset.insert(
		KIND_DINGODB,
		"doc_dir",
		REQUIRE_STRING,
		true,
		nil,
	)

	CONFIG_DINGO_STORE_VECTOR_DIR = itemset.insert(
		KIND_DINGODB,
		"vector_dir",
		REQUIRE_STRING,
		true,
		nil,
	)

	CONFIG_DINGO_EXECUTOR_JAVA_OPTS = itemset.insert(
		KIND_DINGO,
		"java_opts",
		REQUIRE_MAP,
		true,
		func(dc *DeployConfig) interface{} {
			config := dc.GetServiceConfig()
			// filter java prefix config
			java_opts := map[string]interface{}{}
			for k, v := range config {
				if strings.HasPrefix(k, "java.") {
					// subtrim 'java.' prefix
					k = strings.TrimPrefix(k, "java.")
					java_opts[k] = v
				}
			}
			return java_opts
		},
	)

	CONFIG_DINGODB_SERVER_PORT = itemset.insert(
		KIND_DINGODB,
		"port",
		REQUIRE_POSITIVE_INTEGER,
		false,
		func(dc *DeployConfig) interface{} {
			switch dc.GetRole() {
			case ROLE_DINGODB_EXECUTOR:
				return DEFAULT_DINGODB_EXECUTOR_SERVER_PORT
			case ROLE_DINGODB_WEB:
				return DEFAULT_DINGODB_WEB_SERVER_PORT
			case ROLE_DINGODB_PROXY:
				return DEFAULT_DINGODB_PROXY_SERVER_PORT
			}
			return nil
		},
	)

	CONFIG_DINGODB_EXECUTOR_MYSQL_PORT = itemset.insert(
		KIND_DINGODB,
		"mysqlPort",
		REQUIRE_POSITIVE_INTEGER,
		false,
		DEFAULT_DINGODB_EXECUTOR_MYSQL_PORT,
	)

	CONFIG_DINGODB_WEB_EXPORT_PORT = itemset.insert(
		KIND_DINGODB,
		"exportPort",
		REQUIRE_POSITIVE_INTEGER,
		false,
		DEFAULT_DINGODB_WEB_EXPORT_PORT,
	)
)

func (i *item) Key() string {
	return i.key
}

func (itemset *itemSet) insert(kind string, key string, require int, exclude bool, defaultValue interface{}) *item {
	i := &item{kind, key, require, exclude, defaultValue}
	itemset.key2item[key] = i
	itemset.items = append(itemset.items, i)
	return i
}

func (itemset *itemSet) get(key string) *item {
	return itemset.key2item[key]
}

func (itemset *itemSet) getAll() []*item {
	return itemset.items
}
