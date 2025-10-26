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
 * Created Date: 2021-12-23
 * Author: Jingli Chen (Wine93)
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

// __SIGN_BY_WINE93__

package topology

import (
	"fmt"
	"path"
	"strconv"

	"github.com/dingodb/dingoadm/internal/utils"
	"github.com/dingodb/dingoadm/pkg/variable"
)

const (
	// service project layout in container
	LAYOUT_DINGO_ROOT_DIR                    = "/dingo"
	LAYOUT_DINGOFS_ROOT_DIR                  = "/dingofs"
	LAYOUT_DINGOSTORE_ROOT_DIR               = "/opt/dingo-store"
	LAYOUT_DINGOSTORE_BIN_DIR                = "/opt/dingo-store/build/bin"
	LAYOUT_DINGOSTORE_DIST_DIR               = "/opt/dingo-store/dist"
	LAYOUT_DINGDB_DINGO_ROOT_DIR             = "/opt/dingo"
	LAYOUT_CURVEFS_ROOT_DIR                  = "/curvefs"
	LAYOUT_CURVEBS_ROOT_DIR                  = "/curvebs"
	LAYOUT_PLAYGROUND_ROOT_DIR               = "playground"
	LAYOUT_CONF_SRC_DIR                      = "/conf"
	LAYOUT_V2_CONF_SRC_DIR                   = "/conf" // change mdsv2 confv2 to conf
	LAYOUT_SERVICE_BIN_DIR                   = "/sbin"
	LAYOUT_SERVICE_CONF_DIR                  = "/conf"
	LAYOUT_SERVICE_LOGS_DIR                  = "/logs"
	LAYOUT_SERVICE_LOG_DIR                   = "/log"
	LAYOUT_SERVICE_DATA_DIR                  = "/data"
	LAYOUT_TOOLS_DIR                         = "/tools"
	LAYOUT_TOOLS_V2_DIR                      = "/tools-v2"
	LAYOUT_MDSV2_CLIENT_DIR                  = "/mds-client" // change mdsv2-client to mds-client
	LAYOUT_CURVEBS_CHUNKFILE_POOL_DIR        = "chunkfilepool"
	LAYOUT_CURVEBS_COPYSETS_DIR              = "copysets"
	LAYOUT_CURVEBS_RECYCLER_DIR              = "recycler"
	LAYOUT_CURVEBS_TOOLS_CONFIG_SYSTEM_PATH  = "/etc/dingo/tools.conf"
	LAYOUT_CURVEFS_TOOLS_CONFIG_SYSTEM_PATH  = "/etc/dingofs/tools.conf" // v1 tools config path
	LAYOUT_CURVE_TOOLS_V2_CONFIG_SYSTEM_PATH = "/etc/dingo/dingo.yaml"
	// dingo-store coordinator
	LAYOUT_DINGO_COOR_RAFT_DIR = "/coordinator1/data/raft" //TODO: need to be changed
	LAYOUT_DINGO_COOR_DATA_DIR = "/coordinator1/data/db"   //TODO: need to be changed
	LAYOUT_DINGO_COOR_LOG_DIR  = "/coordinator1/log"       //TODO: need to be changed
	// dingo-store store
	LAYOUT_DINGO_STORE_RAFT_DIR = "/store1/data/raft"
	LAYOUT_DINGO_STORE_DATA_DIR = "/store1/data/db"
	LAYOUT_DINGO_STORE_LOG_DIR  = "/store1/log"
	// dingo-store document
	LAYOUT_DINGO_DOCUMENT_DATA_DIR = "/document1/data/db"
	LAYOUT_DINGO_DOCUMENT_LOG_DIR  = "/document1/log"
	LAYOUT_DINGO_DOCUMENT_RAFT_DIR = "/document1/data/raft"
	LAYOUT_DINGO_DOCUMENT_DOC_DIR  = "/document1/data/document_index"
	// dingo-store diskann
	LAYOUT_DINGO_DISKANN_DATA_DIR = "/diskann1/data/diskann"
	LAYOUT_DINGO_DISKANN_LOG_DIR  = "/diskann1/log"
	// dingo-store index
	LAYOUT_DINGO_INDEX_DATA_DIR   = "/index1/data/db"
	LAYOUT_DINGO_INDEX_LOG_DIR    = "/index1/log"
	LAYOUT_DINGO_INDEX_VECTOR_DIR = "/index1/data/vector_index_snapshot"
	LAYOUT_DINGO_INDEX_RAFT_DIR   = "/index1/data/raft"
	// dingo log
	LAYOUT_DINGO_LOG_DIR = "/log"

	LAYOUT_CORE_SYSTEM_DIR = "/core"

	BINARY_CURVEBS_TOOL     = "curvebs-tool"
	BINARY_CURVEBS_FORMAT   = "curve_format"
	BINARY_CURVEFS_TOOL     = "dingo-tool"
	BINARY_DINGO_TOOL_V2    = "dingo"
	BINARY_MDSV2_CLIENT     = "dingo-mds-client"
	METAFILE_CHUNKFILE_POOL = "chunkfilepool.meta"
	METAFILE_CHUNKSERVER_ID = "chunkserver.dat"
)

var (
	DefaultCurveBSDeployConfig = &DeployConfig{kind: KIND_CURVEBS}
	DefaultCurveFSDeployConfig = &DeployConfig{kind: KIND_DINGOFS}

	ServiceConfigs = map[string][]string{
		ROLE_ETCD: {"etcd.conf"},
		// ROLE_MDS_V1:           {"mds.conf"},
		ROLE_CHUNKSERVER:      {"chunkserver.conf", "cs_client.conf", "s3.conf"},
		ROLE_SNAPSHOTCLONE:    {"snapshotclone.conf", "snap_client.conf", "s3.conf", "nginx.conf"},
		ROLE_METASERVER:       {"metaserver.conf"},
		ROLE_COORDINATOR:      {"coordinator-gflags.conf "},
		ROLE_STORE:            {"store-gflags.conf"},
		ROLE_MDS_V2:           {"mds.conf", "mds.template.conf"}, // change dingo-mdsv2.template.conf to mds.template.conf
		ROLE_DINGODB_EXECUTOR: {"executor.yaml"},
		ROLE_DINGODB_WEB:      {"application-web-dev.yaml"},
		ROLE_DINGODB_PROXY:    {"application-proxy-dev.yaml"},
	}
)

func (dc *DeployConfig) get(i *item) interface{} {
	if v, ok := dc.config[i.key]; ok {
		return v
	}

	defaultValue := i.defaultValue
	if defaultValue != nil && utils.IsFunc(defaultValue) {
		return defaultValue.(func(*DeployConfig) interface{})(dc)
	}
	return defaultValue
}

func (dc *DeployConfig) getString(i *item) string {
	v := dc.get(i)
	if v == nil {
		return ""
	}
	return v.(string)
}

func (dc *DeployConfig) getInt(i *item) int {
	v := dc.get(i)
	if v == nil {
		return 0
	}
	// Try direct type assertion first
	if intVal, ok := v.(int); ok {
		return intVal
	}

	// Try converting from string if possible
	if strVal, ok := v.(string); ok {
		if intVal, err := strconv.Atoi(strVal); err == nil {
			return intVal
		}
	}

	// Couldn't convert to int
	return 0

}

func (dc *DeployConfig) getBool(i *item) bool {
	v := dc.get(i)
	if v == nil {
		return false
	}
	return v.(bool)
}

func (dc *DeployConfig) getMap(i *item) map[string]interface{} {
	v := dc.get(i)
	if v == nil {
		return map[string]interface{}{}
	}
	return v.(map[string]interface{})
}

// (1): config property
func (dc *DeployConfig) GetKind() string                     { return dc.kind }
func (dc *DeployConfig) GetId() string                       { return dc.id }
func (dc *DeployConfig) GetParentId() string                 { return dc.parentId }
func (dc *DeployConfig) GetRole() string                     { return dc.role }
func (dc *DeployConfig) GetHost() string                     { return dc.host }
func (dc *DeployConfig) GetHostname() string                 { return dc.hostname }
func (dc *DeployConfig) GetName() string                     { return dc.name }
func (dc *DeployConfig) GetInstances() int                   { return dc.instances }
func (dc *DeployConfig) GetHostSequence() int                { return dc.hostSequence }
func (dc *DeployConfig) GetInstancesSequence() int           { return dc.instancesSequence }
func (dc *DeployConfig) GetServiceConfig() map[string]string { return dc.serviceConfig }
func (dc *DeployConfig) GetVariables() *variable.Variables   { return dc.variables }
func (dc *DeployConfig) GetCtx() *Context                    { return dc.ctx }

// (2): config item
func (dc *DeployConfig) GetPrefix() string         { return dc.getString(CONFIG_PREFIX) }
func (dc *DeployConfig) GetReportUsage() bool      { return dc.getBool(CONFIG_REPORT_USAGE) }
func (dc *DeployConfig) GetContainerImage() string { return dc.getString(CONFIG_CONTAINER_IMAGE) }
func (dc *DeployConfig) GetLogDir() string         { return dc.getString(CONFIG_LOG_DIR) }
func (dc *DeployConfig) GetDataDir() string {
	if dc.GetRole() == ROLE_DINGODB_EXECUTOR || dc.GetRole() == ROLE_DINGODB_WEB || dc.GetRole() == ROLE_DINGODB_PROXY {
		return "-"
	} else if dc.GetRole() == ROLE_MDS_V2 && dc.GetCtx().Lookup(CTX_KEY_MDS_VERSION) == CTX_VAL_MDS_V2 {
		return "-"
	}

	return dc.getString(CONFIG_DATA_DIR)
}
func (dc *DeployConfig) GetSeqOffset() int           { return dc.getInt(CONFIG_SEQ_OFFSET) }
func (dc *DeployConfig) GetSourceCoreDir() string    { return dc.getString(CONFIG_SOURCE_CORE_DIR) }
func (dc *DeployConfig) GetTargetCoreDir() string    { return dc.getString(CONFIG_TARGET_CORE_DIR) }
func (dc *DeployConfig) GetEnv() string              { return dc.getString(CONFIG_ENV) }
func (dc *DeployConfig) GetListenIp() string         { return dc.getString(CONFIG_LISTEN_IP) }
func (dc *DeployConfig) GetListenPort() int          { return dc.getInt(CONFIG_LISTEN_PORT) }
func (dc *DeployConfig) GetListenClientPort() int    { return dc.getInt(CONFIG_LISTEN_CLIENT_PORT) }
func (dc *DeployConfig) GetListenDummyPort() int     { return dc.getInt(CONFIG_LISTEN_DUMMY_PORT) }
func (dc *DeployConfig) GetListenProxyPort() int     { return dc.getInt(CONFIG_LISTEN_PROXY_PORT) }
func (dc *DeployConfig) GetListenExternalIp() string { return dc.getString(CONFIG_LISTEN_EXTERNAL_IP) }
func (dc *DeployConfig) GetCopysets() int            { return dc.getInt(CONFIG_COPYSETS) }
func (dc *DeployConfig) GetS3AccessKey() string      { return dc.getString(CONFIG_S3_ACCESS_KEY) }
func (dc *DeployConfig) GetS3SecretKey() string      { return dc.getString(CONFIG_S3_SECRET_KEY) }
func (dc *DeployConfig) GetS3Address() string        { return dc.getString(CONFIG_S3_ADDRESS) }
func (dc *DeployConfig) GetS3BucketName() string     { return dc.getString(CONFIG_S3_BUCKET_NAME) }
func (dc *DeployConfig) GetEnableRDMA() bool         { return dc.getBool(CONFIG_ENABLE_RDMA) }
func (dc *DeployConfig) GetEnableRenameAt2() bool    { return dc.getBool(CONFIG_ENABLE_RENAMEAT2) }
func (dc *DeployConfig) GetEtcdAuthEnable() bool     { return dc.getBool(CONFIG_ETCD_AUTH_ENABLE) }
func (dc *DeployConfig) GetEtcdAuthUsername() string { return dc.getString(CONFIG_ETCD_AUTH_USERNAME) }
func (dc *DeployConfig) GetEtcdAuthPassword() string { return dc.getString(CONFIG_ETCD_AUTH_PASSWORD) }
func (dc *DeployConfig) GetEnableChunkfilePool() bool {
	return dc.getBool(CONFIG_ENABLE_CHUNKFILE_POOL)
}

func (dc *DeployConfig) GetDingoServerListenHost() string {
	return dc.getString(CONFFIG_DINGO_SERVER_LISTEN_HOST)
}

func (dc *DeployConfig) GetEnableExternalServer() bool {
	return dc.getBool(CONFIG_ENABLE_EXTERNAL_SERVER)
}

func (dc *DeployConfig) GetListenExternalPort() int {
	if dc.GetEnableExternalServer() {
		return dc.getInt(CONFIG_LISTEN_EXTERNAL_PORT)
	}
	return dc.GetListenPort()
}

// GetDingoRaftDir returns the raft directory on the host for the Dingo Store service.
func (dc *DeployConfig) GetDingoRaftDir() string {
	if dc.GetRole() == ROLE_COORDINATOR ||
		dc.GetRole() == ROLE_STORE ||
		dc.GetRole() == ROLE_DINGODB_DOCUMENT ||
		dc.GetRole() == ROLE_DINGODB_INDEX {
		return dc.getString(CONFIG_DINGO_STORE_RAFT_DIR)
	} else {
		return "-"
	}
}

func (dc *DeployConfig) GetDingoStoreDocDir() string {
	if dc.GetRole() == ROLE_DINGODB_DOCUMENT {
		return dc.getString(CONFIG_DINGO_STORE_DOCUMENT_DIR)
	} else {
		return "-"
	}
}

func (dc *DeployConfig) GetDingoStoreVectorDir() string {
	if dc.GetRole() == ROLE_DINGODB_INDEX {
		return dc.getString(CONFIG_DINGO_STORE_VECTOR_DIR)
	} else {
		return "-"
	}
}

func (dc *DeployConfig) GetDingoStoreServerListenHost() string {
	return dc.getString(CONFFIG_DINGO_STORE_SERVER_LISTEN_HOST)
}

func (dc *DeployConfig) GetDingoStoreRaftListenHost() string {
	return dc.getString(CONFFIG_DINGO_STORE_RAFT_LISTEN_HOST)
}

func (dc *DeployConfig) GetDingoServerPort() int {
	return dc.getInt(CONFIG_DINGO_STORE_SERVER_PORT)
}

func (dc *DeployConfig) GetDingoStoreRaftPort() int {
	return dc.getInt(CONFIG_DINGO_STORE_RAFT_PORT)
}

func (dc *DeployConfig) GetDingoDBServerPort() int {
	return dc.getInt(CONFIG_DINGODB_SERVER_PORT)
}

func (dc *DeployConfig) GetDingoDBMySQLPort() int {
	return dc.getInt(CONFIG_DINGODB_EXECUTOR_MYSQL_PORT)
}

func (dc *DeployConfig) GetDingoDBExportPort() int {
	return dc.getInt(CONFIG_DINGODB_WEB_EXPORT_PORT)
}

func (dc *DeployConfig) GetDingoStoreReplicaNum() int {
	return dc.getInt(CONFIG_DINGO_STORE_REPLICA_NUM)
}

func (dc *DeployConfig) GetDingoInstanceId() int {
	return dc.getInt(CONFIG_INSTANCE_START_ID)
}

func (dc *DeployConfig) GetDingoStoreCoordinatorAddr() string {
	return dc.getString(CONFIG_DINGOSTORE_COORDINATOR_ADDR)
}

func (dc *DeployConfig) GetDingoExecutorJavaOpts() map[string]interface{} {
	return dc.getMap(CONFIG_DINGO_EXECUTOR_JAVA_OPTS)
}

//func (dc *DeployConfig) GetDingoServerNum() int {
//	return dc.getInt(CONFIG_DINGO_SERVER_NUM)
//}

// (3): service project layout
/* /curvebs
 *  ├── conf
 *  │   ├── chunkserver.conf
 *  │   ├── etcd.conf
 *  │   ├── mds.conf
 *  │   └── tools.conf
 *  ├── etcd
 *  │   ├── conf
 *  │   ├── data
 *  │   ├── log
 *  │   └── sbin
 *  ├── mds
 *  │   ├── conf
 *  │   ├── data
 *  │   ├── log
 *  │   └── sbin
 *  ├── chunkserver
 *  │   ├── conf
 *  │   ├── data
 *  │   ├── log
 *  │   └── sbin
 *  ├── snapshotclone
 *  │   ├── conf
 *  │   ├── data
 *  │   ├── log
 *  │   └── sbin
 *  └── tools
 *      ├── conf
 *      ├── data
 *      ├── log
 *      └── sbin
 */
type (
	ConfFile struct {
		Name       string
		TargetPath string
		SourcePath string
	}

	// Layout defines the service project container path layout
	Layout struct {
		// project: curvebs/curvefs
		ProjectRootDir string // /curvebs

		PlaygroundRootDir string // /curvebs/playground

		// service
		ServiceRootDir     string // /curvebs/mds
		ServiceBinDir      string // /curvebs/mds/sbin
		ServiceConfDir     string // /curvebs/mds/conf
		ServiceLogDir      string // /curvebs/mds/logs
		ServiceDataDir     string // /curvebs/mds/data
		ServiceConfPath    string // /curvebs/mds/conf/mds.conf
		ServiceConfSrcPath string // /curvebs/conf/mds.conf
		ServiceConfFiles   []ConfFile

		// tools
		ToolsRootDir        string // /curvebs/tools
		ToolsBinDir         string // /curvebs/tools/sbin
		ToolsDataDir        string // /curvebs/tools/data
		ToolsConfDir        string // /curvebs/tools/conf
		ToolsConfPath       string // /curvebs/tools/conf/tools.conf
		ToolsConfSrcPath    string // /curvebs/conf/tools.conf
		ToolsConfSystemPath string // /etc/dingofs/tools.conf
		ToolsBinaryPath     string // /curvebs/tools/sbin/curvebs-tool

		// tools-v2
		ToolsV2BinDir         string // /dingofs/tools-v2/sbin
		ToolsV2ConfDir        string // /dingofs/tools-v2/conf
		ToolsV2ConfSrcPath    string // /dingofs/conf/dingo.yaml
		ToolsV2ConfSrcPath2   string // /dingofs/confv2/dingo.yaml
		ToolsV2ConfSystemPath string // /etc/dingo/dingo.yaml
		ToolsV2BinaryPath     string // /curvebs/tools-v2/sbin/curve

		// mdsv2 client
		MdsV2RootDir        string // /dingofs/mds-client
		MdsV2CliBinDir      string // /dingofs/mds-client/sbin
		MdsV2CliConfDir     string // /dingofs/mds-client/conf
		MdsV2CliConfSrcPath string // /dingofs/mds-client/conf/coor_list
		MdsV2CliBinaryPath  string // /dingofs/mds-client/sbin/dingo-mds-client

		// dingo-store coordinator.template.yaml
		CoordinatorConfSrcPath string // /opt/dingo-store/conf/coordinator.template.yaml
		StoreConfSrcPath       string // /opt/dingo-store/conf/store.template.yaml

		// format
		FormatBinaryPath      string // /curvebs/tools/sbin/curve_format
		ChunkfilePoolRootDir  string // /curvebs/chunkserver/data
		ChunkfilePoolDir      string // /curvebs/chunkserver/data/chunkfilepool
		ChunkfilePoolMetaPath string // /curvebs/chunkserver/data/chunkfilepool.meta

		// dingo-store
		DingoStoreBinDir    string // /opt/dingo-store/build/bin
		DingoStoreRaftDir   string // /opt/dingo-store/xxx/data/raft
		DingoStoreScriptDir string // /opt/dingo-store/scripts

		// dingo-store document
		DingoStoreDocumentDir string // /opt/dingo-store/xxx/data/document_index
		// dingo-store vector
		DingoStoreVectorDir string // /opt/dingo-store/xxx/data/vector_index_snapshot

		//dingo executor
		DingoExecutorBinDir string // /opt/dingo/bin

		// core
		CoreSystemDir string
	}
)

// GetProjectLayout return service project container path layout
func (dc *DeployConfig) GetProjectLayout() Layout {
	kind := dc.GetKind()
	role := dc.GetRole()
	// project
	root := utils.Choose(kind == KIND_CURVEBS, LAYOUT_CURVEBS_ROOT_DIR, LAYOUT_DINGOFS_ROOT_DIR)

	// service
	confSrcDir := root + LAYOUT_CONF_SRC_DIR
	confSrcDirV2 := root + LAYOUT_V2_CONF_SRC_DIR
	serviceRootDir := dc.GetPrefix()
	serviceConfDir := fmt.Sprintf("%s/conf", serviceRootDir)
	serviceConfFiles := []ConfFile{}
	for _, item := range ServiceConfigs[role] {
		sourcePath := fmt.Sprintf("%s/%s", confSrcDir, item)
		targetPath := fmt.Sprintf("%s/%s", serviceConfDir, item)
		if role == ROLE_COORDINATOR ||
			role == ROLE_STORE ||
			role == ROLE_DINGODB_DOCUMENT ||
			role == ROLE_DINGODB_DISKANN ||
			role == ROLE_DINGODB_INDEX ||
			role == ROLE_DINGODB_PROXY ||
			role == ROLE_DINGODB_WEB ||
			role == ROLE_DINGODB_EXECUTOR {
			// dingo-store coordinator/store gflags config
			sourcePath = fmt.Sprintf("%s/%s", serviceConfDir, item)
		} else if role == ROLE_MDS_V2 {
			if dc.ctx.Lookup(CTX_KEY_MDS_VERSION) == CTX_VAL_MDS_V1 {
				// remove "mds.template.conf"
				if item == "mds.template.conf" {
					continue
				}
			}
			if dc.ctx.Lookup(CTX_KEY_MDS_VERSION) == CTX_VAL_MDS_V2 {
				// remove "mds.conf"
				if item == "mds.conf" {
					continue
				}
				sourcePath = fmt.Sprintf("%s/%s", confSrcDirV2, item)
				targetPath = sourcePath
			}
		}
		serviceConfFiles = append(serviceConfFiles, ConfFile{
			Name:       item,
			TargetPath: targetPath,
			SourcePath: sourcePath,
		})
	}

	// tools, change 'dingofs' as root dir
	toolsRootDir := root + LAYOUT_TOOLS_DIR
	toolsBinDir := toolsRootDir + LAYOUT_SERVICE_BIN_DIR
	toolsConfDir := toolsRootDir + LAYOUT_SERVICE_CONF_DIR
	toolsBinaryName := utils.Choose(kind == KIND_CURVEBS, BINARY_CURVEBS_TOOL, BINARY_CURVEFS_TOOL)
	toolsConfSystemPath := LAYOUT_CURVEFS_TOOLS_CONFIG_SYSTEM_PATH

	// tools-v2
	toolsV2RootDir := root + LAYOUT_TOOLS_V2_DIR
	toolsV2BinDir := toolsV2RootDir + LAYOUT_SERVICE_BIN_DIR
	toolsV2BinaryName := BINARY_DINGO_TOOL_V2
	toolsV2ConfSystemPath := LAYOUT_CURVE_TOOLS_V2_CONFIG_SYSTEM_PATH

	// format
	chunkserverDataDir := fmt.Sprintf("%s/%s%s", root, ROLE_CHUNKSERVER, LAYOUT_SERVICE_DATA_DIR)

	serviceLogDir := serviceRootDir + LAYOUT_SERVICE_LOGS_DIR
	serviceDataDir := serviceRootDir + LAYOUT_SERVICE_DATA_DIR
	dingoStoreRaftDir := ""
	dingoStoreDocumentDir := ""
	dingoStoreVectorDir := ""

	switch role {
	case ROLE_COORDINATOR:
		serviceLogDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_COOR_LOG_DIR
		serviceDataDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_COOR_DATA_DIR
		dingoStoreRaftDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_COOR_RAFT_DIR
	case ROLE_STORE:
		serviceLogDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_STORE_LOG_DIR
		serviceDataDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_STORE_DATA_DIR
		dingoStoreRaftDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_STORE_RAFT_DIR
	case ROLE_DINGODB_DOCUMENT:
		serviceLogDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_DOCUMENT_LOG_DIR
		serviceDataDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_DOCUMENT_DATA_DIR
		dingoStoreRaftDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_DOCUMENT_RAFT_DIR
		dingoStoreDocumentDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_DOCUMENT_DOC_DIR
	case ROLE_DINGODB_DISKANN:
		serviceLogDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_DISKANN_LOG_DIR
		serviceDataDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_DISKANN_DATA_DIR
	case ROLE_DINGODB_INDEX:
		serviceLogDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_INDEX_LOG_DIR
		serviceDataDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_INDEX_DATA_DIR
		dingoStoreRaftDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_INDEX_RAFT_DIR
		dingoStoreVectorDir = LAYOUT_DINGOSTORE_DIST_DIR + LAYOUT_DINGO_INDEX_VECTOR_DIR
	case ROLE_DINGODB_EXECUTOR, ROLE_DINGODB_WEB, ROLE_DINGODB_PROXY:
		serviceLogDir = serviceRootDir + LAYOUT_DINGO_LOG_DIR // /opt/dingo/log
	case ROLE_MDS_V2:
		if dc.GetCtx().Lookup(CTX_KEY_MDS_VERSION) == CTX_VAL_MDS_V2 {
			serviceLogDir = serviceRootDir + LAYOUT_SERVICE_LOG_DIR
		}
	default:
		// do nothing
	}

	return Layout{
		// project
		ProjectRootDir: root,

		// playground
		PlaygroundRootDir: path.Join(root, LAYOUT_PLAYGROUND_ROOT_DIR),

		// service
		ServiceRootDir:     serviceRootDir,
		ServiceBinDir:      serviceRootDir + LAYOUT_SERVICE_BIN_DIR,
		ServiceConfDir:     serviceRootDir + LAYOUT_SERVICE_CONF_DIR,
		ServiceLogDir:      serviceLogDir,
		ServiceDataDir:     serviceDataDir,
		ServiceConfPath:    fmt.Sprintf("%s/%s.conf", serviceConfDir, role),
		ServiceConfSrcPath: fmt.Sprintf("%s/%s.conf", confSrcDir, role),
		ServiceConfFiles:   serviceConfFiles,

		// tools
		ToolsRootDir:        toolsRootDir,
		ToolsBinDir:         toolsRootDir + LAYOUT_SERVICE_BIN_DIR,
		ToolsDataDir:        toolsRootDir + LAYOUT_SERVICE_DATA_DIR,
		ToolsConfDir:        toolsRootDir + LAYOUT_SERVICE_CONF_DIR,
		ToolsConfPath:       fmt.Sprintf("%s/tools.conf", toolsConfDir),
		ToolsConfSrcPath:    fmt.Sprintf("%s/tools.conf", confSrcDir),
		ToolsConfSystemPath: toolsConfSystemPath,
		ToolsBinaryPath:     fmt.Sprintf("%s/%s", toolsBinDir, toolsBinaryName),

		// toolsv2
		ToolsV2BinDir:         toolsV2RootDir + LAYOUT_SERVICE_BIN_DIR,
		ToolsV2ConfDir:        toolsV2RootDir + LAYOUT_SERVICE_CONF_DIR,
		ToolsV2ConfSrcPath:    fmt.Sprintf("%s/dingo.yaml", confSrcDir),
		ToolsV2ConfSrcPath2:   fmt.Sprintf("%s/dingo.yaml", confSrcDirV2),
		ToolsV2ConfSystemPath: toolsV2ConfSystemPath,
		ToolsV2BinaryPath:     fmt.Sprintf("%s/%s", toolsV2BinDir, toolsV2BinaryName),

		// mdsv2 client
		MdsV2RootDir:        root + LAYOUT_MDSV2_CLIENT_DIR,
		MdsV2CliBinDir:      root + LAYOUT_MDSV2_CLIENT_DIR + LAYOUT_SERVICE_BIN_DIR,
		MdsV2CliConfDir:     root + LAYOUT_MDSV2_CLIENT_DIR + LAYOUT_SERVICE_CONF_DIR,
		MdsV2CliConfSrcPath: fmt.Sprintf("%s/coor_list", root+LAYOUT_MDSV2_CLIENT_DIR+LAYOUT_SERVICE_CONF_DIR), // /dingofs/mds-client/conf/coor_list
		MdsV2CliBinaryPath:  fmt.Sprintf("%s/%s", root+LAYOUT_MDSV2_CLIENT_DIR+LAYOUT_SERVICE_BIN_DIR, BINARY_MDSV2_CLIENT),

		// format
		FormatBinaryPath:      fmt.Sprintf("%s/%s", toolsBinDir, BINARY_CURVEBS_FORMAT),
		ChunkfilePoolRootDir:  chunkserverDataDir,
		ChunkfilePoolDir:      fmt.Sprintf("%s/%s", chunkserverDataDir, LAYOUT_CURVEBS_CHUNKFILE_POOL_DIR),
		ChunkfilePoolMetaPath: fmt.Sprintf("%s/%s", chunkserverDataDir, METAFILE_CHUNKFILE_POOL),

		// dingo-store
		DingoStoreBinDir:    LAYOUT_DINGOSTORE_BIN_DIR,
		DingoStoreRaftDir:   dingoStoreRaftDir,
		DingoStoreScriptDir: LAYOUT_DINGOSTORE_ROOT_DIR + "/scripts",

		// dingo-store document
		DingoStoreDocumentDir: dingoStoreDocumentDir,
		// dingo-store vector
		DingoStoreVectorDir: dingoStoreVectorDir,

		// dingo executor
		DingoExecutorBinDir: serviceRootDir + "/bin",

		// core
		CoreSystemDir: LAYOUT_CORE_SYSTEM_DIR,
	}
}

func GetProjectLayout(kind, role string) Layout {
	dc := DeployConfig{kind: kind, role: role}
	return dc.GetProjectLayout()
}

func GetCurveBSProjectLayout() Layout {
	return DefaultCurveBSDeployConfig.GetProjectLayout()
}

func GetDingoFSProjectLayout() Layout {
	return DefaultCurveFSDeployConfig.GetProjectLayout()
}
