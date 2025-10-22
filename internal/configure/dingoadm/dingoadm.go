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

package dingoadm

import (
	"fmt"
	"os"
	"regexp"

	"github.com/dingodb/dingoadm/internal/build"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/viper"
)

/*
 * [defaults]
 * log_level = error
 * sudo_alias = "sudo"
 * timeout = 180
 *
 * [ssh_connections]
 * retries = 3
 * timeout = 10
 *
 * [database]
 * url = "sqlite:///home/curve/.curveadm/data/curveadm.db"
 */
const (
	KEY_LOG_LEVEL    = "log_level"
	KEY_SUDO_ALIAS   = "sudo_alias"
	KEY_ENGINE       = "engine"
	KEY_TIMEOUT      = "timeout"
	KEY_AUTO_UPGRADE = "auto_upgrade"
	KEY_SSH_RETRIES  = "retries"
	KEY_SSH_TIMEOUT  = "timeout"
	KEY_DB_URL       = "url"

	// rqlite://127.0.0.1:4000
	// sqlite:///home/curve/.curveadm/data/curveadm.db
	REGEX_DB_URL = "^(sqlite|rqlite)://(.+)$"
	DB_SQLITE    = "sqlite"
	DB_RQLITE    = "rqlite"

	WITHOUT_SUDO = " "
)

type (
	DingoAdmConfig struct {
		LogLevel    string
		SudoAlias   string
		Engine      string
		Timeout     int
		AutoUpgrade bool
		SSHRetries  int
		SSHTimeout  int
		DBUrl       string
	}

	DingoAdm struct {
		Defaults       map[string]interface{} `mapstructure:"defaults"`
		SSHConnections map[string]interface{} `mapstructure:"ssh_connections"`
		DataBase       map[string]interface{} `mapstructure:"database"`
	}
)

var (
	GlobalDingoAdmConfig *DingoAdmConfig

	SUPPORT_LOG_LEVEL = map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
)

func ReplaceGlobals(cfg *DingoAdmConfig) {
	GlobalDingoAdmConfig = cfg
}

func newDefault() *DingoAdmConfig {
	home, _ := os.UserHomeDir()
	cfg := &DingoAdmConfig{
		LogLevel:    "error",
		SudoAlias:   "sudo",
		Engine:      "docker",
		Timeout:     180,
		AutoUpgrade: true,
		SSHRetries:  3,
		SSHTimeout:  10,
		DBUrl:       fmt.Sprintf("sqlite://%s/.dingoadm/data/dingoadm.db", home),
	}
	return cfg
}

// TODO(P2): using ItemSet to check value type
func requirePositiveInt(k string, v interface{}) (int, error) {
	num, ok := utils.Str2Int(v.(string))
	if !ok {
		return 0, errno.ERR_CONFIGURE_VALUE_REQUIRES_INTEGER.
			F("%s: %v", k, v)
	} else if num <= 0 {
		return 0, errno.ERR_CONFIGURE_VALUE_REQUIRES_POSITIVE_INTEGER.
			F("%s: %v", k, v)
	}
	return num, nil
}

func requirePositiveBool(k string, v interface{}) (bool, error) {
	yes, ok := utils.Str2Bool(v.(string))
	if !ok {
		return false, errno.ERR_CONFIGURE_VALUE_REQUIRES_BOOL.
			F("%s: %v", k, v)
	}
	return yes, nil
}

func parseDefaultsSection(cfg *DingoAdmConfig, defaults map[string]interface{}) error {
	if defaults == nil {
		return nil
	}

	for k, v := range defaults {
		switch k {
		// log_level
		case KEY_LOG_LEVEL:
			if !SUPPORT_LOG_LEVEL[v.(string)] {
				return errno.ERR_UNSUPPORT_DINGOADM_LOG_LEVEL.
					F("%s: %s", KEY_LOG_LEVEL, v.(string))
			}
			cfg.LogLevel = v.(string)

		// sudo_alias
		case KEY_SUDO_ALIAS:
			cfg.SudoAlias = v.(string)

		// container engine
		case KEY_ENGINE:
			cfg.Engine = v.(string)

		// timeout
		case KEY_TIMEOUT:
			num, err := requirePositiveInt(KEY_TIMEOUT, v)
			if err != nil {
				return err
			}
			cfg.Timeout = num

		// auto upgrade
		case KEY_AUTO_UPGRADE:
			yes, err := requirePositiveBool(KEY_AUTO_UPGRADE, v)
			if err != nil {
				return err
			}
			cfg.AutoUpgrade = yes

		default:
			return errno.ERR_UNSUPPORT_DINGOADM_CONFIGURE_ITEM.
				F("%s: %s", k, v)
		}
	}

	return nil
}

func parseConnectionSection(cfg *DingoAdmConfig, connection map[string]interface{}) error {
	if connection == nil {
		return nil
	}

	for k, v := range connection {
		switch k {
		// ssh_retries
		case KEY_SSH_RETRIES:
			num, err := requirePositiveInt(KEY_SSH_RETRIES, v)
			if err != nil {
				return err
			}
			cfg.SSHRetries = num

		// ssh_timeout
		case KEY_SSH_TIMEOUT:
			num, err := requirePositiveInt(KEY_SSH_TIMEOUT, v)
			if err != nil {
				return err
			}
			cfg.SSHTimeout = num

		default:
			return errno.ERR_UNSUPPORT_DINGOADM_CONFIGURE_ITEM.
				F("%s: %s", k, v)
		}
	}

	return nil
}

func parseDatabaseSection(cfg *DingoAdmConfig, database map[string]interface{}) error {
	if database == nil {
		return nil
	}

	for k, v := range database {
		switch k {
		// database url
		case KEY_DB_URL:
			dbUrl := v.(string)
			pattern := regexp.MustCompile(REGEX_DB_URL)
			mu := pattern.FindStringSubmatch(dbUrl)
			if len(mu) == 0 {
				return errno.ERR_UNSUPPORT_DINGOADM_DATABASE_URL.F("url: %s", dbUrl)
			}
			cfg.DBUrl = dbUrl

		default:
			return errno.ERR_UNSUPPORT_DINGOADM_CONFIGURE_ITEM.
				F("%s: %s", k, v)
		}
	}

	return nil
}

type sectionParser struct {
	parser  func(*DingoAdmConfig, map[string]interface{}) error
	section map[string]interface{}
}

func ParseDingoAdmConfig(filename string) (*DingoAdmConfig, error) {
	cfg := newDefault()
	if !utils.PathExist(filename) {
		build.DEBUG(build.DEBUG_CURVEADM_CONFIGURE, cfg)
		return cfg, nil
	}

	// parse dingoadm config
	parser := viper.New()
	parser.SetConfigFile(filename)
	parser.SetConfigType("ini")
	err := parser.ReadInConfig()
	if err != nil {
		return nil, errno.ERR_PARSE_DINGOADM_CONFIGURE_FAILED.E(err)
	}

	global := &DingoAdm{}
	err = parser.Unmarshal(global)
	if err != nil {
		return nil, errno.ERR_PARSE_DINGOADM_CONFIGURE_FAILED.E(err)
	}

	items := []sectionParser{
		{parseDefaultsSection, global.Defaults},
		{parseConnectionSection, global.SSHConnections},
		{parseDatabaseSection, global.DataBase},
	}
	for _, item := range items {
		err := item.parser(cfg, item.section)
		if err != nil {
			return nil, err
		}
	}

	build.DEBUG(build.DEBUG_CURVEADM_CONFIGURE, cfg)
	return cfg, nil
}

func (cfg *DingoAdmConfig) GetLogLevel() string  { return cfg.LogLevel }
func (cfg *DingoAdmConfig) GetTimeout() int      { return cfg.Timeout }
func (cfg *DingoAdmConfig) GetAutoUpgrade() bool { return cfg.AutoUpgrade }
func (cfg *DingoAdmConfig) GetSSHRetries() int   { return cfg.SSHRetries }
func (cfg *DingoAdmConfig) GetSSHTimeout() int   { return cfg.SSHTimeout }
func (cfg *DingoAdmConfig) GetEngine() string    { return cfg.Engine }
func (cfg *DingoAdmConfig) GetSudoAlias() string {
	if len(cfg.SudoAlias) == 0 {
		return WITHOUT_SUDO
	}
	return cfg.SudoAlias
}

func (cfg *DingoAdmConfig) GetDBUrl() string {
	return cfg.DBUrl
}

func (cfg *DingoAdmConfig) GetDBPath() string {
	pattern := regexp.MustCompile(REGEX_DB_URL)
	mu := pattern.FindStringSubmatch(cfg.DBUrl)
	if len(mu) == 0 || mu[1] != DB_SQLITE {
		return ""
	}
	return mu[2]
}
