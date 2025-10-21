/*
*  Copyright (c) 2023 NetEase Inc.
*  Copyright (c) 2025 dingodb.com.
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
* Created Date: 2023-04-21
* Author: wanghai (SeanHai)
*
* Project: Dingoadm
* Author: jackblack369 (Dongwei)
 */

package monitor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/scripts"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/internal/task/task/common"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/pkg/variable"
)

const (
	TOOL_SYS_PATH                  = "/usr/bin/curve_ops_tool"
	MONITOR_CONF_PATH              = "monitor"
	PROMETHEUS_CONTAINER_CONF_PATH = "/etc/prometheus"
	GRAFANA_CONTAINER_PATH         = "/etc/grafana/grafana.ini"
	DASHBOARD_CONTAINER_PATH       = "/etc/grafana/provisioning/dashboards"
	GRAFANA_DATA_SOURCE_PATH       = "/etc/grafana/provisioning/datasources/all.yml"
	CURVE_MANAGER_CONF_PATH        = "/curve-manager/conf/pigeon.yaml"
	DINGO_TOOL_SRC_PATH            = "/dingofs/conf/dingo.yaml"
	DINGO_TOOL_DEST_PATH           = "/etc/dingo/dingo.yaml"
	ORIGIN_MONITOR_PATH            = "/dingofs/monitor"
)

func syncPrometheusUid(cfg *configure.MonitorConfig, dingoadm cli.DingoAdm) step.LambdaType {
	return func(ctx *context.Context) error {
		var prometheusInfo string
		// fetch prometheus info retry 10 times
		for i := 0; i < 10; i++ {
			curlStep := &step.Curl{
				Url:         fmt.Sprintf("-u admin:admin http://localhost:%d/api/datasources", cfg.GetListenPort()),
				Insecure:    true,
				Out:         &prometheusInfo,
				Silent:      true,
				ExecOptions: dingoadm.ExecOptions(),
			}
			curlStep.Execute(ctx)
			if len(prometheusInfo) > 0 {
				// try parse prometheus info
				var arr []map[string]interface{}
				if err := json.Unmarshal([]byte(prometheusInfo), &arr); err == nil && len(arr) > 0 {
					if v, ok := arr[0]["uid"].(string); ok {
						// *prometheusUid = v
						sedStep := &step.Command{
							Command:     combineSedCMD(cfg, v),
							ExecOptions: dingoadm.ExecOptions(),
						}
						sedStep.Execute(ctx)
						return nil
					}
					return nil
				}
			}
			time.Sleep(3 * time.Second)
		}
		return fmt.Errorf("failed to fetch prometheus info")
	}
}

func combineSedCMD(cfg *configure.MonitorConfig, prometheusUid string) string {
	return fmt.Sprintf(`sed -i 's/${PROMETHEUS_UID}/%s/g' %s/dashboards/server_metric_zh.json`, prometheusUid, cfg.GetProvisionDir())
}

func MutateTool(vars *variable.Variables, delimiter string) step.Mutate {
	return func(in, key, value string) (out string, err error) {
		if len(key) == 0 {
			out = in
			return
		}

		// replace variable
		value, err = vars.Rendering(value)
		if err != nil {
			return
		}

		out = fmt.Sprintf("%s%s%s", key, delimiter, value)
		return
	}
}

func getNodeExporterAddrs(hosts []string, port int) string {
	endpoint := []string{}
	for _, item := range hosts {
		endpoint = append(endpoint, fmt.Sprintf("'%s:%d'", item, port))
	}
	return fmt.Sprintf("[%s]", strings.Join(endpoint, ","))
}

func NewSyncConfigTask(dingoadm *cli.DingoAdm, cfg *configure.MonitorConfig) (*task.Task, error) {
	serviceId := dingoadm.GetServiceId(cfg.GetId())
	containerId, err := dingoadm.GetContainerId(serviceId)
	if IsSkip(cfg, []string{ROLE_MONITOR_CONF, ROLE_NODE_EXPORTER}) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	role, host := cfg.GetRole(), cfg.GetHost()
	hc, err := dingoadm.GetHost(host)
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s role=%s containerId=%s",
		cfg.GetHost(), cfg.GetRole(), tui.TrimContainerId(containerId))
	t := task.NewTask("Sync Config", subname, hc.GetSSHConfig())
	// add step to task
	var out string
	t.AddStep(&step.ListContainers{ // gurantee container exist
		ShowAll:     true,
		Format:      `"{{.ID}}"`,
		Filter:      fmt.Sprintf("id=%s", containerId),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: common.CheckContainerExist(cfg.GetHost(), cfg.GetRole(), containerId, &out),
	})

	// confServiceId = dingoadm.GetServiceId(fmt.Sprintf("%s_%s", ROLE_MONITOR_SYNC, cfg.GetHost()))
	// confContainerId, err := dingoadm.GetContainerId(serviceId)

	if role == ROLE_PROMETHEUS {

		// replace prometheus/prometheus.yml port info
		sedCMD := fmt.Sprintf(`sed -i 's/localhost:[0-9]*/localhost:%d/g' %s/prometheus.yml`, cfg.GetListenPort(), cfg.GetConfDir())
		t.AddStep(&step.Command{
			Command:     sedCMD,
			Out:         &out,
			ExecOptions: dingoadm.ExecOptions(),
		})

		// replace node exporter addrs
		nodeHosts := cfg.GetNodeIps()
		nodePort := cfg.GetNodeListenPort()
		nodeExporterAddrs := getNodeExporterAddrs(nodeHosts, nodePort)

		// install sync_prometheus.sh
		t.AddStep(&step.InstallFile{
			HostDestPath: fmt.Sprintf("%s/sync_prometheus.sh", cfg.GetConfDir()),
			Content:      &scripts.SYNC_PROMETHEUS,
			ExecOptions:  dingoadm.ExecOptions(),
		})

		t.AddStep(&step.Command{
			Command:     fmt.Sprintf("bash %s/sync_prometheus.sh %s/prometheus.yml %s", cfg.GetConfDir(), cfg.GetConfDir(), nodeExporterAddrs),
			Out:         &out,
			ExecOptions: dingoadm.ExecOptions(),
		})

	} else if role == ROLE_GRAFANA {

		// replace grafana/provisioning/datasources/all.yml port info
		sedPortCMD := fmt.Sprintf(`sed -i 's/localhost:[0-9]*/localhost:%d/g' %s/datasources/all.yml`, cfg.GetPrometheusListenPort(), cfg.GetProvisionDir())
		t.AddStep(&step.Command{
			Command:     sedPortCMD,
			Out:         &out,
			ExecOptions: dingoadm.ExecOptions(),
		})

	} else if role == ROLE_MONITOR_SYNC {

		confID := cfg.GetServiceConfig()[configure.KEY_ORIGIN_CONFIG_ID].(string)
		confServiceId := dingoadm.GetServiceId(confID)
		confContainerId, err := dingoadm.GetContainerId(confServiceId)
		if err != nil {
			return nil, err
		}

		t.AddStep(&step.TrySyncFile{ // sync tools-v2 config
			ContainerSrcId:    &confContainerId,
			ContainerSrcPath:  DINGO_TOOL_DEST_PATH,
			ContainerDestId:   &containerId,
			ContainerDestPath: DINGO_TOOL_DEST_PATH,
			KVFieldSplit:      common.CONFIG_DELIMITER_COLON,
			Mutate:            MutateTool(cfg.GetVariables(), common.CONFIG_DELIMITER_COLON),
			ExecOptions:       dingoadm.ExecOptions(),
		})

		hostMonitorDir := cfg.GetDataDir()
		t.AddStep(&common.Step2CopyFilesFromContainer{ // copy monitor directory
			ContainerId:   confContainerId,
			Files:         &[]string{ORIGIN_MONITOR_PATH},
			HostDestDir:   hostMonitorDir,
			ExcludeParent: true,
			Dingoadm:      dingoadm,
		})

		t.AddStep(&step.InstallFile{ // install start_monitor_sync script
			HostDestPath: hostMonitorDir + "/start_monitor_sync.sh",
			Content:      &scripts.START_MONITOR_SYNC,
			ExecOptions:  dingoadm.ExecOptions(),
		})

		t.AddStep(&step.Command{
			Command:     fmt.Sprintf("chmod +x %s/start_monitor_sync.sh", hostMonitorDir),
			Out:         &out,
			ExecOptions: dingoadm.ExecOptions(),
		})
	}
	return t, nil
}

func NewSyncGrafanaDashboardTask(dingoadm *cli.DingoAdm, cfg *configure.MonitorConfig) (*task.Task, error) {
	serviceId := dingoadm.GetServiceId(cfg.GetId())
	containerId, err := dingoadm.GetContainerId(serviceId)
	if IsSkip(cfg, []string{ROLE_MONITOR_CONF, ROLE_NODE_EXPORTER}) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	_, host := cfg.GetRole(), cfg.GetHost()
	hc, err := dingoadm.GetHost(host)
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s role=%s containerId=%s",
		cfg.GetHost(), cfg.GetRole(), tui.TrimContainerId(containerId))
	t := task.NewTask("Sync Grafana Dashboard", subname, hc.GetSSHConfig())
	// add step to task
	var out string
	t.AddStep(&step.ListContainers{ // gurantee container exist
		ShowAll:     true,
		Format:      `"{{.ID}}"`,
		Filter:      fmt.Sprintf("id=%s", containerId),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: common.CheckContainerExist(cfg.GetHost(), cfg.GetRole(), containerId, &out),
	})

	t.AddStep(&step.InstallFile{ // install server_metric_zh.json
		HostDestPath: fmt.Sprintf("%s/dashboards/%s", cfg.GetProvisionDir(), "server_metric_zh.json"),
		Content:      &scripts.GRAFANA_SERVER_METRIC,
		ExecOptions:  dingoadm.ExecOptions(),
	})

	// wait for grafana service started
	t.AddStep(&step.Lambda{
		//Lambda: wait(30),
		Lambda: syncPrometheusUid(cfg, *dingoadm),
	})

	return t, nil
}
