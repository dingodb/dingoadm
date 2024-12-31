package configure

import (
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/viper"
)

const (
	KEY_GATEWAY_S3_ROOT_USER                = "s3.root_user"
	KEY_GATEWAY_S3_ROOT_PASSWORD            = "s3.root_password"
	KEY_GATEWAY_LISTEN_PORT                 = "gateway.listen_port"
	KEY_GATEWAY_CONSOLE_PORT                = "gateway.console_port"
	KEY_DINGOFS_MDS_ADDRS                   = "dingofs.mdsaddr"
	KEY_GATEWAY_CONTAINER_IMAGE             = "container_image"
	KEY_GATEWAY_LOG_DIR                     = "log_dir"
	KEY_GATEWAY_DATA_DIR                    = "data_dir"
	KEY_GATEWAY_CORE_DIR                    = "core_dir"
	DEFAULT_DINGOFS_GATEWAY_CONTAINER_IMAGE = "dingodatabase/dingofs:latest"
)

type (
	GatewayConfig struct {
		configMap map[string]interface{}
		data      string // configure file content
	}
)

func NewGatewayConfig(config map[string]interface{}) (*GatewayConfig, error) {
	gc := &GatewayConfig{
		configMap: config,
	}

	return gc, nil
}

func ParseGatewayConfig(filename string) (*GatewayConfig, error) {
	// 1. read file content
	data, err := utils.ReadFile(filename)
	if err != nil {
		return nil, errno.ERR_PARSE_GATEWAY_CONFIGURE_FAILED.E(err)
	}

	// 2. new parser
	parser := viper.NewWithOptions(viper.KeyDelimiter("::"))
	parser.SetConfigFile(filename)
	parser.SetConfigType("yaml")
	err = parser.ReadInConfig()
	if err != nil {
		return nil, errno.ERR_PARSE_GATEWAY_CONFIGURE_FAILED.E(err)
	}

	// 3. parse
	m := map[string]interface{}{}
	err = parser.Unmarshal(&m)
	if err != nil {
		return nil, errno.ERR_PARSE_GATEWAY_CONFIGURE_FAILED.E(err)
	}

	// 4. new config
	cfg, err := NewGatewayConfig(m)
	if err != nil {
		return nil, err
	}

	cfg.data = data
	return cfg, nil
}

func (gc *GatewayConfig) getString(key string) string {
	v := gc.configMap[strings.ToLower(key)]
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func (gc *GatewayConfig) getBool(key string) bool {
	v := gc.configMap[strings.ToLower(key)]
	if v == nil {
		return false
	}
	return v.(bool)
}

func (gc *GatewayConfig) GetS3RootUser() string { return gc.getString(KEY_GATEWAY_S3_ROOT_USER) }
func (gc *GatewayConfig) GetS3RootPassword() string {
	return gc.getString(KEY_GATEWAY_S3_ROOT_PASSWORD)
}

func (gc *GatewayConfig) GetListenPort() string {
	return gc.getString(KEY_GATEWAY_LISTEN_PORT)
}

func (gc *GatewayConfig) GetConsolePort() string {
	return gc.getString(KEY_GATEWAY_CONSOLE_PORT)
}

func (gc *GatewayConfig) GetDataDir() string { return gc.getString(KEY_GATEWAY_DATA_DIR) }
func (gc *GatewayConfig) GetLogDir() string  { return gc.getString(KEY_GATEWAY_LOG_DIR) }
func (gc *GatewayConfig) GetCoreDir() string { return gc.getString(KEY_GATEWAY_CORE_DIR) }
func (gc *GatewayConfig) GetData() string    { return gc.data }
func (gc *GatewayConfig) GetContainerImage() string {
	containerImage := gc.getString(KEY_GATEWAY_CONTAINER_IMAGE)
	if len(containerImage) == 0 {
		containerImage = DEFAULT_DINGOFS_GATEWAY_CONTAINER_IMAGE
	}
	return containerImage
}

func (gc *GatewayConfig) GetDingofsMDSAddr() string {
	return gc.getString(KEY_DINGOFS_MDS_ADDRS)
}
