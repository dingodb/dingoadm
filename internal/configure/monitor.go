/*
*  Copyright (c) 2023 NetEase Inc.
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
* Project: Curveadm
* Created Date: 2023-04-17
* Author: wanghai (SeanHai)
 */

package configure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/hosts"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/spf13/viper"
)

const (
	ROLE_NODE_EXPORTER = "node_exporter"
	ROLE_PROMETHEUS    = "prometheus"
	ROLE_GRAFANA       = "grafana"
	ROLE_MONITOR_CONF  = "monitor_conf"

	KEY_HOST              = "host"
	KEY_LISTEN_PORT       = "listen_port"
	KEY_RETENTION_TIME    = "retention.time"
	KEY_RETENTION_SIZE    = "retention.size"
	KEY_PROMETHEUS_TARGET = "target"
	KEY_GRAFANA_USER      = "username"
	KEY_GRAFANA_PASSWORD  = "password"

	KEY_NODE_IPS         = "node_ips"
	KRY_NODE_LISTEN_PORT = "node_listen_port"
	KEY_PROMETHEUS_IP    = "prometheus_listen_ip"
	KEY_PROMETHEUS_PORT  = "prometheus_listen_port"
)

type monitor struct {
	Host         string                 `mapstructure:"host"`
	NodeExporter map[string]interface{} `mapstructure:"node_exporter"`
	Prometheus   map[string]interface{} `mapstructure:"prometheus"`
	Grafana      map[string]interface{} `mapstructure:"grafana"`
}

type MonitorConfig struct {
	kind   string
	id     string // role_host
	role   string
	host   string
	config map[string]interface{}
	ctx    *topology.Context
}

type serviceTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

type FilterMonitorOption struct {
	Id   string
	Role string
	Host string
}

func (m *MonitorConfig) getString(data *map[string]interface{}, key string) string {
	v := (*data)[strings.ToLower(key)]
	if v == nil {
		return ""
	}
	return v.(string)
}

func (m *MonitorConfig) getStrings(data *map[string]interface{}, key string) []string {
	v := (*data)[strings.ToLower(key)]
	if v == nil {
		return []string{}
	}
	return v.([]string)
}

func (m *MonitorConfig) getInt(data *map[string]interface{}, key string) int {
	v := (*data)[strings.ToLower(key)]
	if v == nil {
		return -1
	}
	return v.(int)
}

func (m *MonitorConfig) GetKind() string {
	return m.kind
}

func (m *MonitorConfig) GetId() string {
	return m.id
}

func (m *MonitorConfig) GetRole() string {
	return m.role
}

func (m *MonitorConfig) GetHost() string {
	return m.host
}

func (m *MonitorConfig) GetNodeIps() []string {
	return m.getStrings(&m.config, KEY_NODE_IPS)
}

func (m *MonitorConfig) GetNodeListenPort() int {
	return m.getInt(&m.config, KRY_NODE_LISTEN_PORT)
}

func (m *MonitorConfig) GetPrometheusListenPort() int {
	return m.getInt(&m.config, KEY_PROMETHEUS_PORT)
}

func (m *MonitorConfig) GetImage() string {
	return m.getString(&m.config, KEY_CONTAINER_IMAGE)
}

func (m *MonitorConfig) GetListenPort() int {
	return m.getInt(&m.config, KEY_LISTEN_PORT)
}

func (m *MonitorConfig) GetDataDir() string {
	return m.getString(&m.config, KEY_DATA_DIR)
}

func (m *MonitorConfig) GetLogDir() string {
	return m.getString(&m.config, KEY_LOG_DIR)
}

func (m *MonitorConfig) GetPrometheusRetentionTime() string {
	return m.getString(&m.config, KEY_RETENTION_TIME)
}

func (m *MonitorConfig) GetPrometheusRetentionSize() string {
	return m.getString(&m.config, KEY_RETENTION_SIZE)
}

func (m *MonitorConfig) GetPrometheusTarget() string {
	return m.getString(&m.config, KEY_PROMETHEUS_TARGET)
}

func (m *MonitorConfig) GetPrometheusIp() string {
	return m.getString(&m.config, KEY_PROMETHEUS_IP)
}

func (m *MonitorConfig) GetGrafanaUser() string {
	return m.getString(&m.config, KEY_GRAFANA_USER)
}

func (m *MonitorConfig) GetGrafanaPassword() string {
	return m.getString(&m.config, KEY_GRAFANA_PASSWORD)
}

func getHost(c *monitor, role string) string {
	h := c.Host
	switch role {
	case ROLE_NODE_EXPORTER:
		if _, ok := c.NodeExporter[KEY_HOST]; ok {
			return c.NodeExporter[KEY_HOST].(string)
		}
		c.NodeExporter[KEY_HOST] = h
	case ROLE_PROMETHEUS:
		if _, ok := c.Prometheus[KEY_HOST]; ok {
			return c.Prometheus[KEY_HOST].(string)
		}
		c.Prometheus[KEY_HOST] = h
	case ROLE_GRAFANA:
		if _, ok := c.Grafana[KEY_HOST]; ok {
			return c.Grafana[KEY_HOST].(string)
		}
		c.Grafana[KEY_HOST] = h
	}
	return h
}

func parsePrometheusTarget(dcs []*topology.DeployConfig) (string, error) {
	targets := []serviceTarget{}
	tMap := make(map[string]serviceTarget)
	for _, dc := range dcs {
		role := dc.GetRole()
		ip := dc.GetListenIp()
		var item string
		switch role {
		case topology.ROLE_ETCD:
			item = fmt.Sprintf("%s:%d", ip, dc.GetListenClientPort())
		case topology.ROLE_MDS,
			topology.ROLE_CHUNKSERVER,
			topology.ROLE_METASERVER:
			item = fmt.Sprintf("%s:%d", ip, dc.GetListenPort())
		case topology.ROLE_SNAPSHOTCLONE:
			item = fmt.Sprintf("%s:%d", ip, dc.GetListenDummyPort())
		}
		if _, ok := tMap[role]; ok {
			t := tMap[role]
			t.Targets = append(t.Targets, item)
			tMap[role] = t
		} else {
			tMap[role] = serviceTarget{
				Labels:  map[string]string{"job": role},
				Targets: []string{item},
			}
		}
	}
	for _, v := range tMap {
		targets = append(targets, v)
	}
	target, err := json.Marshal(targets)
	if err != nil {
		return "", errno.ERR_PARSE_PROMETHEUS_TARGET_FAILED.E(err)
	}
	return string(target), nil
}

func ParseMonitorConfig(curveadm *cli.DingoAdm, filename string, data string, hs []string,
	hostIps []string, dcs []*topology.DeployConfig) (
	[]*MonitorConfig, error) {
	parser := viper.NewWithOptions(viper.KeyDelimiter("::"))
	parser.SetConfigType("yaml")
	if len(data) != 0 && data != common.CLEANED_MONITOR_CONF {
		if err := parser.ReadConfig(bytes.NewBuffer([]byte(data))); err != nil {
			return nil, errno.ERR_PARSE_MONITOR_CONFIGURE_FAILED.E(err)
		}
	} else if len(filename) != 0 {
		parser.SetConfigFile(filename)
		if err := parser.ReadInConfig(); err != nil {
			return nil, errno.ERR_PARSE_MONITOR_CONFIGURE_FAILED.E(err)
		}
	} else {
		return nil, errno.ERR_PARSE_MONITOR_CONFIGURE_FAILED
	}

	config := monitor{}
	if err := parser.Unmarshal(&config); err != nil {
		return nil, errno.ERR_PARSE_MONITOR_CONFIGURE_FAILED.E(err)
	}

	// get host -> hostname(ip)
	ctx := topology.NewContext()
	hcs, err := hosts.ParseHosts(curveadm.Hosts())
	if err != nil {
		return nil, err
	}
	for _, hc := range hcs {
		ctx.Add(hc.GetHost(), hc.GetHostname())
	}

	mkind := dcs[0].GetKind()
	mconfImage := dcs[0].GetContainerImage()
	roles := []string{}
	switch {
	case config.NodeExporter != nil:
		roles = append(roles, ROLE_NODE_EXPORTER)
		fallthrough
	case config.Prometheus != nil:
		roles = append(roles, ROLE_PROMETHEUS)
		fallthrough
	case config.Grafana != nil:
		roles = append(roles, ROLE_GRAFANA)
	}
	ret := []*MonitorConfig{}
	for _, role := range roles {
		host := getHost(&config, role)
		switch role {
		case ROLE_PROMETHEUS:
			target, err := parsePrometheusTarget(dcs)
			if err != nil {
				return nil, err
			}
			if config.NodeExporter != nil {
				config.Prometheus[KEY_NODE_IPS] = hostIps
				config.Prometheus[KRY_NODE_LISTEN_PORT] = config.NodeExporter[KEY_LISTEN_PORT]
			}
			config.Prometheus[KEY_PROMETHEUS_TARGET] = target
			ret = append(ret, &MonitorConfig{
				kind:   mkind,
				id:     fmt.Sprintf("%s_%s", role, host),
				role:   role,
				host:   host,
				config: config.Prometheus,
				ctx:    ctx,
			})
		case ROLE_GRAFANA:
			if config.Prometheus != nil {
				config.Grafana[KEY_PROMETHEUS_PORT] = config.Prometheus[KEY_LISTEN_PORT]
				config.Grafana[KEY_PROMETHEUS_IP] = ctx.Lookup(config.Prometheus[KEY_HOST].(string))
			}
			ret = append(ret, &MonitorConfig{
				kind:   mkind,
				id:     fmt.Sprintf("%s_%s", role, host),
				role:   role,
				host:   host,
				config: config.Grafana,
				ctx:    ctx,
			}, &MonitorConfig{
				kind: mkind,
				id:   fmt.Sprintf("%s_%s", ROLE_MONITOR_CONF, host),
				role: ROLE_MONITOR_CONF,
				host: host,
				config: map[string]interface{}{
					KEY_CONTAINER_IMAGE: mconfImage,
				},
				ctx: ctx,
			})
		case ROLE_NODE_EXPORTER:
			for _, h := range hs {
				ret = append(ret, &MonitorConfig{
					kind:   mkind,
					id:     fmt.Sprintf("%s_%s", role, h),
					role:   role,
					host:   h,
					config: config.NodeExporter,
					ctx:    ctx,
				})
			}
		}
	}
	return ret, nil
}

func FilterMonitorConfig(curveadm *cli.DingoAdm, mcs []*MonitorConfig,
	options FilterMonitorOption) []*MonitorConfig {
	ret := []*MonitorConfig{}
	for _, mc := range mcs {
		mcId := mc.GetId()
		role := mc.GetRole()
		host := mc.GetHost()
		serviceId := curveadm.GetServiceId(mcId)
		if (options.Id == "*" || options.Id == serviceId) &&
			(options.Role == "*" || options.Role == role) &&
			(options.Host == "*" || options.Host == host) {
			ret = append(ret, mc)
		}
	}
	return ret
}
