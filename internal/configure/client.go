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
 * Created Date: 2022-01-09
 * Author: Jingli Chen (Wine93)
 */

package configure

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/internal/build"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/utils"
	log "github.com/dingodb/dingoadm/pkg/log/glg"
	"github.com/dingodb/dingoadm/pkg/variable"
	"github.com/spf13/viper"
)

const (
	KEY_KIND                     = "kind"
	KEY_CONTAINER_IMAGE          = "container_image"
	KEY_LOG_DIR                  = "log_dir"
	KEY_DATA_DIR                 = "data_dir"
	KEY_CORE_DIR                 = "core_dir"
	KEY_CACHE_DIR                = "mount_dirs"
	KEY_CURVEBS_LISTEN_MDS_ADDRS = "mds.listen.addr"
	KEY_CURVEFS_LISTEN_MDS_ADDRS = "mdsOpt.rpcRetryOpt.addrs"
	KEY_CONTAINER_PID            = "container_pid"
	KEY_ENVIRONMENT              = "env"

	KEY_CLIENT_S3_ACCESS_KEY  = "s3.ak"
	KEY_CLIENT_S3_SECRET_KEY  = "s3.sk"
	KEY_CLIENT_S3_ADDRESS     = "s3.endpoint"
	KEY_CLIENT_S3_BUCKET_NAME = "s3.bucket_name"

	KEY_QUOTA_CAPACITY = "quota.capacity"
	KEY_QUOTA_INODES   = "quota.inodes"

	DEFAULT_CORE_LOCATE_DIR = "/core"
)

const (
	DEFAULT_CURVEBS_CLIENT_CONTAINER_IMAGE = "opencurvedocker/curvebs:v1.2"
	DEFAULT_CURVEFS_CLIENT_CONTAINER_IMAGE = "dingodatabase/dingofs:latest"
)

var (
	excludeClientConfig = map[string]bool{
		KEY_CONTAINER_IMAGE: true,
		KEY_LOG_DIR:         true,
		KEY_DATA_DIR:        true,
		KEY_CORE_DIR:        true,
		KEY_CACHE_DIR:       true,
		KEY_CONTAINER_PID:   true,
		KEY_ENVIRONMENT:     true,
	}

	LAYOUT_CURVEBS_ROOT_DIR = topology.GetCurveBSProjectLayout().ProjectRootDir
	LAYOUT_DINGOFS_ROOT_DIR = topology.GetDingoFSProjectLayout().ProjectRootDir
)

type (
	Client struct {
		Config map[string]interface{}
	}

	ClientConfig struct {
		config        map[string]interface{}
		serviceConfig map[string]string
		variables     *variable.Variables
		data          string // configure file content
	}
)

func NewClientConfig(config map[string]interface{}) (*ClientConfig, error) {
	serviceConfig := map[string]string{}
	for k, v := range config {
		value, ok := utils.All2Str(v)
		if !ok {
			return nil, errno.ERR_UNSUPPORT_CLIENT_CONFIGURE_VALUE_TYPE.
				F("%s: %v", k, v)
		}
		if !excludeClientConfig[k] { // TODO(P0): check bool or integer
			serviceConfig[k] = value
		}
	}

	vars := variable.NewVariables()
	vars.Register(variable.Variable{Name: "prefix", Value: "/curvebs/nebd"})
	err := vars.Build()
	if err != nil {
		log.Error("Build variables failed", log.Field("Error", err))
		return nil, errno.ERR_RESOLVE_CLIENT_VARIABLE_FAILED.E(err)
	}

	cc := &ClientConfig{
		config:        config,
		serviceConfig: serviceConfig,
		variables:     vars,
	}

	kind := cc.GetKind()
	field := utils.Choose(kind == topology.KIND_CURVEBS,
		KEY_CURVEBS_LISTEN_MDS_ADDRS, KEY_CURVEFS_LISTEN_MDS_ADDRS)
	if cc.GetKind() != topology.KIND_CURVEBS && kind != topology.KIND_CURVEFS && kind != topology.KIND_DINGOFS {
		return nil, errno.ERR_UNSUPPORT_CLIENT_CONFIGURE_KIND.
			F("kind: %s", kind)
	} else if len(cc.GetClusterMDSAddr()) == 0 {
		return nil, errno.ERR_INVALID_CLUSTER_LISTEN_MDS_ADDRESS.
			F("%s: %s", field, cc.GetClusterMDSAddr())
	}
	return cc, nil
}

func ParseClientCfg(data string) (*ClientConfig, error) {
	parser := viper.NewWithOptions(viper.KeyDelimiter("::"))
	parser.SetConfigType("yaml")
	err := parser.ReadConfig(bytes.NewBuffer([]byte(data)))
	if err != nil {
		return nil, errno.ERR_PARSE_CLIENT_CONFIGURE_FAILED.E(err)
	}

	config := map[string]interface{}{}
	if err := parser.Unmarshal(&config); err != nil {
		return nil, errno.ERR_PARSE_CLIENT_CONFIGURE_FAILED.E(err)
	}
	build.DEBUG(build.DEBUG_CLIENT_CONFIGURE, config)
	return NewClientConfig(config)
}

func ParseClientConfig(filename string) (*ClientConfig, error) {
	// 1. read file content
	data, err := utils.ReadFile(filename)
	if err != nil {
		return nil, errno.ERR_PARSE_CLIENT_CONFIGURE_FAILED.E(err)
	}

	// 2. new parser
	parser := viper.NewWithOptions(viper.KeyDelimiter("::"))
	parser.SetConfigFile(filename)
	parser.SetConfigType("yaml")
	err = parser.ReadInConfig()
	if err != nil {
		return nil, errno.ERR_PARSE_CLIENT_CONFIGURE_FAILED.E(err)
	}

	// 3. parse
	m := map[string]interface{}{}
	err = parser.Unmarshal(&m)
	if err != nil {
		return nil, errno.ERR_PARSE_CLIENT_CONFIGURE_FAILED.E(err)
	}

	// 4. new config
	cfg, err := NewClientConfig(m)
	if err != nil {
		return nil, err
	}

	cfg.data = data
	return cfg, nil
}

func (cc *ClientConfig) getString(key string) string {
	v := cc.config[strings.ToLower(key)]
	if v == nil {
		return ""
	}
	return v.(string)
}

func (cc *ClientConfig) getBool(key string) bool {
	v := cc.config[strings.ToLower(key)]
	if v == nil {
		return false
	}
	return v.(bool)
}

func (cc *ClientConfig) getDigital(key string) int {
	v := cc.config[strings.ToLower(key)]
	if v == nil {
		return 0
	}
	return v.(int)
}

func (cc *ClientConfig) GetKind() string                     { return cc.getString(KEY_KIND) }
func (cc *ClientConfig) GetDataDir() string                  { return cc.getString(KEY_DATA_DIR) }
func (cc *ClientConfig) GetLogDir() string                   { return cc.getString(KEY_LOG_DIR) }
func (cc *ClientConfig) GetCoreDir() string                  { return cc.getString(KEY_CORE_DIR) }
func (cc *ClientConfig) GetMapperCacheDir() string           { return cc.getString(KEY_CACHE_DIR) }
func (cc *ClientConfig) GetS3AccessKey() string              { return cc.getString(KEY_CLIENT_S3_ACCESS_KEY) }
func (cc *ClientConfig) GetS3SecretKey() string              { return cc.getString(KEY_CLIENT_S3_SECRET_KEY) }
func (cc *ClientConfig) GetS3Address() string                { return cc.getString(KEY_CLIENT_S3_ADDRESS) }
func (cc *ClientConfig) GetS3BucketName() string             { return cc.getString(KEY_CLIENT_S3_BUCKET_NAME) }
func (cc *ClientConfig) GetContainerPid() string             { return cc.getString(KEY_CONTAINER_PID) }
func (cc *ClientConfig) GetEnvironments() string             { return cc.getString(KEY_ENVIRONMENT) }
func (cc *ClientConfig) GetQuotaCapacity() int               { return cc.getDigital(KEY_QUOTA_CAPACITY) }
func (cc *ClientConfig) GetQuotaInodes() int                 { return cc.getDigital(KEY_QUOTA_INODES) }
func (cc *ClientConfig) GetCoreLocateDir() string            { return DEFAULT_CORE_LOCATE_DIR }
func (cc *ClientConfig) GetData() string                     { return cc.data }
func (cc *ClientConfig) GetServiceConfig() map[string]string { return cc.serviceConfig }
func (cc *ClientConfig) GetVariables() *variable.Variables   { return cc.variables }
func (cc *ClientConfig) GetContainerImage() string {
	containerImage := cc.getString(KEY_CONTAINER_IMAGE)
	if len(containerImage) == 0 {
		containerImage = utils.Choose(cc.GetKind() == topology.KIND_CURVEBS,
			DEFAULT_CURVEBS_CLIENT_CONTAINER_IMAGE,
			DEFAULT_CURVEFS_CLIENT_CONTAINER_IMAGE)
	}
	return containerImage
}

func (cc *ClientConfig) GetClusterMDSAddr() string {
	if cc.GetKind() == topology.KIND_CURVEBS {
		return cc.getString(KEY_CURVEBS_LISTEN_MDS_ADDRS)
	}
	return cc.getString(KEY_CURVEFS_LISTEN_MDS_ADDRS)
}

// wrapper interface: curvefs client related
func GetFSProjectRoot() string {
	return LAYOUT_DINGOFS_ROOT_DIR
}

func GetBSProjectRoot() string {
	return LAYOUT_CURVEBS_ROOT_DIR
}

func GetFSClientPrefix() string {
	return fmt.Sprintf("%s/client", LAYOUT_DINGOFS_ROOT_DIR)
}

func GetFSClientConfPath() string {
	return fmt.Sprintf("%s/client/conf/client.conf", LAYOUT_DINGOFS_ROOT_DIR)
}

func GetFSClientMountPath(subPath string) string {
	// return fmt.Sprintf("%s/client/mnt%s", LAYOUT_DINGOFS_ROOT_DIR, subPath)
	return fmt.Sprintf("/host%s", subPath)
}
