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
 * Project: dingoadm
 * Created Date: 2021-10-15
 * Author: Jingli Chen (Wine93)
 */

// __SIGN_BY_WINE93__

package common

import (
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/task/scripts"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
)

const (
	CONFIG_DELIMITER_ASSIGN = "="
	CONFIG_DELIMITER_COLON  = ": "

	CURVE_CRONTAB_FILE      = "/tmp/curve_crontab"
	CONFIG_DEFAULT_ENV_FILE = "/etc/profile"
	STORE_BUILD_BIN_DIR     = "/opt/dingo-store/build/bin"
)

func NewMutate(dc *topology.DeployConfig, delimiter string, forceRender bool) step.Mutate {
	serviceConfig := dc.GetServiceConfig()
	return func(in, key, value string) (out string, err error) {
		if len(key) == 0 {
			out = in
			if forceRender { // only for nginx.conf
				out, err = dc.GetVariables().Rendering(in)
			}
			return
		}

		muteKey := strings.TrimSpace(key)
		if dc.GetRole() == topology.ROLE_COORDINATOR || dc.GetRole() == topology.ROLE_STORE {
			// key is like -xxx , replace  '-' to 'gflags.'
			if strings.HasPrefix(key, comm.STORE_GFLAGS_PREFIX) {
				muteKey = fmt.Sprintf("gflags.%s", strings.TrimPrefix(key, comm.STORE_GFLAGS_PREFIX))
			}
		} else if dc.GetRole() == topology.ROLE_MDS_V2 {
			// key is like --xxx , trim '--'
			if strings.HasPrefix(key, comm.MDSV2_CONFIG_PREFIX) {
				muteKey = strings.TrimPrefix(key, comm.MDSV2_CONFIG_PREFIX)
			}
		}

		// replace config
		v, ok := serviceConfig[strings.ToLower(muteKey)]
		if ok {
			value = v
		}

		// replace variable
		value, err = dc.GetVariables().Rendering(value)
		if err != nil {
			return
		}

		out = fmt.Sprintf("%s%s%s", key, delimiter, value)
		return
	}
}

func newCrontab(uuid string, dc *topology.DeployConfig, reportScriptPath string) string {
	var period, command string
	if dc.GetReportUsage() == true {
		period = func(minute, hour, day, month, week string) string {
			return fmt.Sprintf("%s %s %s %s %s", minute, hour, day, month, week)
		}("0", "*", "*", "*", "*") // every hour

		command = func(format string, args ...interface{}) string {
			return fmt.Sprintf(format, args...)
		}("bash %s %s %s %s", reportScriptPath, dc.GetKind(), uuid, dc.GetRole())
	}

	return fmt.Sprintf("%s %s\n", period, command)
}

func syncJavaOpts(java_opts map[string]interface{}, hostSyncJavaOptsScriptPath, hostStartExecutorPath string) string {
	cmd_envs := ""
	if len(java_opts) == 0 {
		return cmd_envs
	}
	// iterate config map item to env
	for k, v := range java_opts {
		// config command env in command line
		cmd_envs += fmt.Sprintf("%s=%s ", k, v)
	}

	command := fmt.Sprintf("%s bash %s %s", cmd_envs, hostSyncJavaOptsScriptPath, hostStartExecutorPath)
	return command
}

func NewSyncConfigTask(dingoadm *cli.DingoAdm, dc *topology.DeployConfig) (*task.Task, error) {
	if dc.GetRole() == topology.ROLE_MDSV2_CLI {
		skipTmp := dingoadm.MemStorage().Get(comm.KEY_SKIP_MDSV2_CLI)
		if skipTmp != nil && skipTmp.(bool) {
			return nil, nil
		}
	}

	serviceId := dingoadm.GetServiceId(dc.GetId())
	containerId, err := dingoadm.GetContainerId(serviceId)
	if dingoadm.IsSkip(dc) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	hc, err := dingoadm.GetHost(dc.GetHost())
	if err != nil {
		return nil, err
	}

	// new task
	subname := fmt.Sprintf("host=%s role=%s containerId=%s",
		dc.GetHost(), dc.GetRole(), tui.TrimContainerId(containerId))
	t := task.NewTask("Sync Config", subname, hc.GetSSHConfig())

	// add step to task
	var out string
	layout := dc.GetProjectLayout()
	role := dc.GetRole()

	delimiter := CONFIG_DELIMITER_ASSIGN
	if role == topology.ROLE_ETCD || role == topology.ROLE_DINGODB_EXECUTOR ||
		role == topology.ROLE_DINGODB_WEB || role == topology.ROLE_DINGODB_PROXY {
		delimiter = CONFIG_DELIMITER_COLON
	}

	t.AddStep(&step.ListContainers{ // gurantee container exist
		ShowAll:     true,
		Format:      `"{{.ID}}"`,
		Filter:      fmt.Sprintf("id=%s", containerId),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: CheckContainerExist(dc.GetHost(), dc.GetRole(), containerId, &out),
	})

	if dc.GetKind() == topology.KIND_DINGOFS || dc.GetKind() == topology.KIND_DINGODB {
		for _, conf := range layout.ServiceConfFiles {
			t.AddStep(&step.SyncFile{ // sync service config, e.g. dingo-mdsv2.template.conf
				ContainerSrcId:    &containerId,
				ContainerSrcPath:  conf.SourcePath,
				ContainerDestId:   &containerId,
				ContainerDestPath: conf.TargetPath,
				KVFieldSplit:      delimiter,
				Mutate:            NewMutate(dc, delimiter, conf.Name == "nginx.conf"),
				SerivceConfig:     dc.GetServiceConfig(),
				ExecOptions:       dingoadm.ExecOptions(),
			})
		}

		if dc.GetRole() == topology.ROLE_DINGODB_DOCUMENT ||
			dc.GetRole() == topology.ROLE_DINGODB_DISKANN ||
			dc.GetRole() == topology.ROLE_DINGODB_INDEX ||
			dc.GetRole() == topology.ROLE_DINGODB_PROXY ||
			dc.GetRole() == topology.ROLE_DINGODB_WEB {
			// return directly, no more config to sync
			return t, nil
		}

		if dc.GetRole() == topology.ROLE_COORDINATOR || dc.GetRole() == topology.ROLE_STORE {
			// sync check_store_health.sh
			checkStoreScript := scripts.CHECK_STORE_HEALTH
			checkStoreScriptPath := fmt.Sprintf("%s/%s", layout.DingoStoreScriptDir, topology.SCRIPT_CHECK_STORE_HEALTH) // /opt/dingo-store/scripts
			t.AddStep(&step.InstallFile{                                                                                 // install create_mdsv2_tables.sh script
				ContainerId:       &containerId,
				ContainerDestPath: checkStoreScriptPath,
				Content:           &checkStoreScript,
				ExecOptions:       dingoadm.ExecOptions(),
			})

			return t, nil
		} else if dc.GetRole() == topology.ROLE_MDSV2_CLI {
			// sync create_mdsv2_tables.sh
			createTablesScript := scripts.CREATE_MDSV2_TABLES
			createTablesScriptPath := fmt.Sprintf("%s/%s", layout.MdsV2CliBinDir, topology.SCRIPT_CREATE_MDSV2_TABLES) // /dingofs/mdsv2-client/sbin
			// createTablesScriptPath := fmt.Sprintf("%s/create_mdsv2_tables.sh", STORE_BUILD_BIN_DIR) // /opt/dingo-store/build/bin
			t.AddStep(&step.InstallFile{ // install create_mdsv2_tables.sh script
				ContainerId:       &containerId,
				ContainerDestPath: createTablesScriptPath,
				Content:           &createTablesScript,
				ExecOptions:       dingoadm.ExecOptions(),
			})

		} else if dc.GetRole() == topology.ROLE_DINGODB_EXECUTOR {
			java_opts := dc.GetDingoExecutorJavaOpts()
			if len(java_opts) == 0 {
				// if no java opts config, return directly
				return t, nil
			}
			// sync executor java opts config /opt/dingo/bin/start-executor.sh
			syncJavaOptsScript := scripts.SYNC_JAVA_OPTS
			// containerSyncJavaOptsScriptPath := fmt.Sprintf("%s/%s", layout.DingoExecutorBinDir, topology.SCRIPT_SYNC_JAVA_OPTS)
			hostSyncJavaOptsScriptPath := fmt.Sprintf("%s/%s", TEMP_DIR, topology.SCRIPT_SYNC_JAVA_OPTS)
			containerStartExecutorPath := fmt.Sprintf("%s/%s", layout.DingoExecutorBinDir, topology.SCRIPT_START_EXECUTOR)
			hostStartExecutorPath := fmt.Sprintf("%s/%s", TEMP_DIR, topology.SCRIPT_START_EXECUTOR)
			t.AddStep(&step.InstallFile{ // install sync_java_opts.sh on local script
				HostDestPath: hostSyncJavaOptsScriptPath,
				Content:      &syncJavaOptsScript,
				ExecOptions:  dingoadm.ExecOptions(),
			})

			t.AddStep(&step.CopyFromContainer{ // copy container /opt/dingo/bin/start-executor.sh to host
				ContainerId:      containerId,
				ContainerSrcPath: containerStartExecutorPath,
				HostDestPath:     hostStartExecutorPath,
				ExecOptions:      dingoadm.ExecOptions(),
			})

			t.AddStep(&step.Command{
				Command:     syncJavaOpts(java_opts, hostSyncJavaOptsScriptPath, hostStartExecutorPath),
				Out:         &out,
				ExecOptions: dingoadm.ExecOptions(),
			})

			t.AddStep(&step.CopyIntoContainer{ // copy host start-executor.sh to container
				HostSrcPath:       hostStartExecutorPath,
				ContainerId:       containerId,
				ContainerDestPath: containerStartExecutorPath,
				ExecOptions:       dingoadm.ExecOptions(),
			})

		} else {
			containerToolsSrcPath := layout.ToolsV2ConfSrcPath
			if dc.GetRole() == topology.ROLE_MDS_V2 {
				containerToolsSrcPath = layout.ToolsV2ConfSrcPath2
			}
			t.AddStep(&step.TrySyncFile{ // sync tools-v2 config
				ContainerSrcId:    &containerId,
				ContainerSrcPath:  containerToolsSrcPath,
				ContainerDestId:   &containerId,
				ContainerDestPath: layout.ToolsV2ConfSystemPath,
				KVFieldSplit:      CONFIG_DELIMITER_COLON,
				Mutate:            NewMutate(dc, CONFIG_DELIMITER_COLON, false),
				ExecOptions:       dingoadm.ExecOptions(),
			})

			// install report script and crontab file
			reportScript := scripts.REPORT
			reportScriptPath := fmt.Sprintf("%s/report.sh", layout.ToolsV2BinDir) // v1: ToolsBinDir, v2: ToolsV2BinDir
			crontab := newCrontab(dingoadm.ClusterUUId(), dc, reportScriptPath)
			t.AddStep(&step.InstallFile{ // install report script
				ContainerId:       &containerId,
				ContainerDestPath: reportScriptPath,
				Content:           &reportScript,
				ExecOptions:       dingoadm.ExecOptions(),
			})
			t.AddStep(&step.InstallFile{ // install crontab file
				ContainerId:       &containerId,
				ContainerDestPath: CURVE_CRONTAB_FILE,
				Content:           &crontab,
				ExecOptions:       dingoadm.ExecOptions(),
			})
		}

	}

	return t, nil
}
