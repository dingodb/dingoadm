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
 * Modified Date: 2025-06-20
 * Modified By: Dongwei
 */

package fs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/task/context"
	"github.com/dingodb/dingoadm/internal/task/scripts"
	"github.com/dingodb/dingoadm/internal/task/step"
	"github.com/dingodb/dingoadm/internal/task/task"
	"github.com/dingodb/dingoadm/internal/task/task/checker"
	"github.com/dingodb/dingoadm/internal/utils"
)

const (
	FORMAT_MOUNT_OPTION = "type=bind,source=%s,target=%s,bind-propagation=rshared"

	KEY_CURVEBS_CLUSTER = "curvebs.cluster"

	CURVEBS_CONF_PATH = "/etc/dingo/client.conf"
)

type (
	MountOptions struct {
		Host        string
		MountFSName string
		MountFSType string
		MountPoint  string
	}

	step2InsertClient struct {
		curveadm    *cli.DingoAdm
		options     MountOptions
		config      *configure.ClientConfig
		containerId *string
	}

	AuxInfo struct {
		FSName     string `json:"fsname"`
		MountPoint string `json:"mount_point,"`
		Config     string `json:"config,omitempty"` // TODO(P1)
	}
)

var (
	// TODO(P1): use template
	FORMAT_FUSE_ARGS = []string{
		"-f",
		"-o default_permissions",
		"-o allow_other",
		"-o fsname=%s", // fsname
		"-o fstype=%s", // v1: s3, mdsv2: vfs
		"-o user=dingofs",
		"-o conf=%s", // config path
		"%s",         // mount path
	}
)

func getMountCommand(cc *configure.ClientConfig, mountFSName string, mountFSType string, mountPoint string, useNewDingo bool) string {
	format := strings.Join(FORMAT_FUSE_ARGS, " ")
	// TODO change mountFSType
	fuseArgs := fmt.Sprintf(format, mountFSName, mountFSType, configure.GetFSClientConfPath(), configure.GetFSClientMountPath(mountPoint))

	//fmt.Printf("docker bootstrap command: /client.sh %s %s --role=client --args='%s' --capacity=%d --inodes=%d\n", mountFSName, mountFSType, fuseArgs, cc.GetQuotaCapacity(), cc.GetQuotaInodes())
	if useNewDingo {
		return fmt.Sprintf("/client.sh %s %s --role=client --args='%s' --capacity=%d --inodes=%d --new-dingo", mountFSName, mountFSType, fuseArgs, cc.GetQuotaCapacity(), cc.GetQuotaInodes())
	} else {
		return fmt.Sprintf("/client.sh %s %s --role=client --args='%s' --capacity=%d --inodes=%d", mountFSName, mountFSType, fuseArgs, cc.GetQuotaCapacity(), cc.GetQuotaInodes())
	}

}

func getMountVolumes(cc *configure.ClientConfig) []step.Volume {
	volumes := []step.Volume{}
	prefix := configure.GetFSClientPrefix()
	logDir := cc.GetLogDir()
	dataDir := cc.GetDataDir()
	coreDir := cc.GetCoreDir()
	cacheDir := cc.GetMapperCacheDir()

	if len(logDir) > 0 {
		volumes = append(volumes, step.Volume{
			HostPath:      logDir,
			ContainerPath: fmt.Sprintf("%s/logs", prefix),
		})
	}

	if len(dataDir) > 0 {
		volumes = append(volumes, step.Volume{
			HostPath:      dataDir,
			ContainerPath: fmt.Sprintf("%s/data", prefix),
		})
	}

	if len(coreDir) > 0 {
		volumes = append(volumes, step.Volume{
			HostPath:      coreDir,
			ContainerPath: cc.GetCoreLocateDir(),
		})
	}

	if len(cacheDir) > 0 {
		// host_path_1:container_path_1;host_path_2:container_path_2;host_path_3:container_path_3
		for hostPath, containerPath := range parseMountPaths(cacheDir) {
			volumes = append(volumes, step.Volume{
				HostPath:      hostPath,
				ContainerPath: containerPath,
			})
		}
	}

	return volumes
}

func newMutate(cc *configure.ClientConfig, delimiter string) step.Mutate {
	serviceConfig := cc.GetServiceConfig()
	return func(in, key, value string) (out string, err error) {
		if len(key) == 0 {
			out = in
			return
		}

		// replace config
		v, ok := serviceConfig[strings.ToLower(key)]
		if ok {
			value = v
		}

		out = fmt.Sprintf("%s%s%s", key, delimiter, value)
		return
	}
}

func newCurveBSMutate(cc *configure.ClientConfig, delimiter string) step.Mutate {
	serviceConfig := cc.GetServiceConfig()

	// we need `curvebs.cluster` if fstype is volume
	if serviceConfig[KEY_CURVEBS_CLUSTER] == "" {
		return func(in, key, value string) (out string, err error) {
			err = errors.New("need `curvebs.cluster` if fstype is `volume`")
			return
		}
	}

	bsClientItems := map[string]string{
		"mds.listen.addr": KEY_CURVEBS_CLUSTER,
	}
	bsClientFixedOptions := map[string]string{
		"mds.registerToMDS":     "false",
		"global.logging.enable": "false",
	}
	return func(in, key, value string) (out string, err error) {
		if len(key) == 0 {
			out = in
			return
		}

		_, ok := bsClientFixedOptions[key]
		if ok {
			value = bsClientFixedOptions[key]
		} else {
			replaceKey := key
			if bsClientItems[key] != "" {
				replaceKey = bsClientItems[key]
			}

			v, ok := serviceConfig[replaceKey]
			if ok {
				value = v
			}
		}

		out = fmt.Sprintf("%s%s%s", key, delimiter, value)
		return
	}
}

func newToolsMutate(cc *configure.ClientConfig, delimiter string) step.Mutate {
	clientConfig := cc.GetServiceConfig()
	tools2client := map[string]string{
		"mdsAddr":       "mdsOpt.rpcRetryOpt.addrs",
		"volumeCluster": KEY_CURVEBS_CLUSTER,
	}
	return func(in, key, value string) (out string, err error) {
		if len(key) == 0 {
			out = in
			return
		}
		replaceKey := key
		if tools2client[key] != "" {
			replaceKey = tools2client[key]
		}
		v, ok := clientConfig[strings.ToLower(replaceKey)]
		if ok {
			value = v
		}
		out = fmt.Sprintf("%s%s%s", key, delimiter, value)
		return
	}
}

func newToolsV2Mutate(cc *configure.ClientConfig, delimiter string) step.Mutate {
	clientConfig := cc.GetServiceConfig()
	// mapping client config to dingo config
	tools2client := map[string]string{
		"mdsAddr":         "mdsOpt.rpcRetryOpt.addrs",
		"ak":              "s3.ak",
		"sk":              "s3.sk",
		"endpoint":        "s3.endpoint",
		"bucketname":      "s3.bucket_name",
		"storagetype":     "storage.type",
		"username":        "rados.username",
		"key":             "rados.key",
		"mon":             "rados.mon",
		"poolname":        "rados.poolname",
		"mds_api_version": "mds.api_version",
	}
	return func(in, key, value string) (out string, err error) {
		if len(key) == 0 {
			out = in
			return
		}
		trimKey := strings.TrimSpace(key)
		replaceKey := trimKey
		if tools2client[trimKey] != "" {
			replaceKey = tools2client[trimKey]
		}
		v, ok := clientConfig[strings.ToLower(replaceKey)]
		if ok {
			value = v
		}
		out = fmt.Sprintf("%s%s%s", key, delimiter, value)
		return
	}
}

func mountPoint2ContainerName(mountPoint string) string {
	return fmt.Sprintf("dingofs-filesystem-%s", utils.MD5Sum(mountPoint))
}

func checkMountStatus(mountPoint, name string, out *string) step.LambdaType {
	return func(ctx *context.Context) error {
		if *out == name {
			return errno.ERR_FS_PATH_ALREADY_MOUNTED.F("mountPath: %s", mountPoint)
		}
		return nil
	}
}

func getEnvironments(cc *configure.ClientConfig) []string {
	envs := []string{
		"LD_PRELOAD=/usr/local/lib/libjemalloc.so",
	}
	env := cc.GetEnvironments()
	if len(env) > 0 {
		envs = append(envs, strings.Split(env, " ")...)
	}
	return envs
}

func (s *step2InsertClient) Execute(ctx *context.Context) error {
	config := s.config
	curveadm := s.curveadm
	options := s.options
	fsId := curveadm.GetFilesystemId(options.Host, options.MountPoint)

	auxInfo := &AuxInfo{
		FSName:     options.MountFSName,
		MountPoint: options.MountPoint,
	}
	bytes, err := json.Marshal(auxInfo)
	if err != nil {
		return errno.ERR_ENCODE_VOLUME_INFO_TO_JSON_FAILED.E(err)
	}

	err = curveadm.Storage().InsertClient(fsId, config.GetKind(),
		options.Host, *s.containerId, string(bytes))
	if err != nil {
		return errno.ERR_INSERT_CLIENT_FAILED.E(err)
	}

	err = curveadm.Storage().InsertClientConfig(fsId, config.GetData())
	if err != nil {
		return errno.ERR_INSERT_CLIENT_CONFIG_FAILED.E(err)
	}

	return nil
}

func checkStartContainerStatus(success *bool, out *string) step.LambdaType {
	return func(ctx *context.Context) error {
		if *success {
			return nil
		} else if strings.Contains(*out, "CREATEFS FAILED") {
			return errno.ERR_CREATE_FILESYSTEM_FAILED
		}
		return errno.ERR_MOUNT_FILESYSTEM_FAILED.S(*out)
	}
}

func NewMountFSTask(dingoadm *cli.DingoAdm, cc *configure.ClientConfig) (*task.Task, error) {
	options := dingoadm.MemStorage().Get(comm.KEY_MOUNT_OPTIONS).(MountOptions)
	useNewDingo := dingoadm.MemStorage().Get(comm.KEY_USE_NEW_DINGO).(bool)
	hc, err := dingoadm.GetHost(options.Host)
	if err != nil {
		return nil, err
	}

	// new task
	mountPoint := options.MountPoint
	mountFSName := options.MountFSName
	mountFSType := options.MountFSType
	subname := fmt.Sprintf("mountFSName=%s mountFSType=%s mountPoint=%s", mountFSName, mountFSType, mountPoint)
	t := task.NewTask("Mount FileSystem", subname, hc.GetSSHConfig())

	// add step to task
	var containerId, out string
	var success bool
	root := configure.GetFSProjectRoot()
	prefix := configure.GetFSClientPrefix()
	containerMountPath := configure.GetFSClientMountPath(mountPoint)
	containerName := mountPoint2ContainerName(mountPoint)
	createfsScript := scripts.CREATE_FS
	createfsScriptPath := "/client.sh"

	t.AddStep(&step.EngineInfo{
		Success:     &success,
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: checker.CheckEngineInfo(options.Host, dingoadm.ExecOptions().ExecWithEngine, &success, &out),
	})
	t.AddStep(&step.ListContainers{
		ShowAll:     true,
		Format:      "'{{.Names}}'",
		Filter:      fmt.Sprintf("name=%s", containerName),
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: checkMountStatus(mountPoint, containerName, &out),
	})
	useLocalImage := dingoadm.MemStorage().Get(comm.KEY_USE_LOCAL_IMAGE).(bool)
	if !useLocalImage {
		t.AddStep(&step.PullImage{
			Image:       cc.GetContainerImage(),
			ExecOptions: dingoadm.ExecOptions(),
		})
	}
	t.AddStep(&step.CreateContainer{
		Image:             cc.GetContainerImage(),
		Command:           getMountCommand(cc, mountFSName, mountFSType, mountPoint, useNewDingo),
		Entrypoint:        "/bin/bash",
		Envs:              getEnvironments(cc),
		Init:              true,
		Name:              mountPoint2ContainerName(mountPoint),
		Mount:             fmt.Sprintf(FORMAT_MOUNT_OPTION, mountPoint, containerMountPath),
		Volumes:           getMountVolumes(cc),
		Devices:           []string{"/dev/fuse"},
		SecurityOptions:   []string{"apparmor:unconfined"},
		LinuxCapabilities: []string{"SYS_ADMIN"},
		Ulimits:           []string{"core=-1"},
		Pid:               cc.GetContainerPid(),
		Privileged:        true,
		Out:               &containerId,
		ExecOptions:       dingoadm.ExecOptions(),
	})
	t.AddStep(&step2InsertClient{
		curveadm:    dingoadm,
		options:     options,
		config:      cc,
		containerId: &containerId,
	})

	if mountFSType == "volume" {
		t.AddStep(&step.SyncFile{ // sync volume client config
			ContainerSrcId:    &containerId,
			ContainerSrcPath:  fmt.Sprintf("%s/conf/curvebs-client.conf", root),
			ContainerDestId:   &containerId,
			ContainerDestPath: CURVEBS_CONF_PATH,
			KVFieldSplit:      comm.CLIENT_CONFIG_DELIMITER,
			Mutate:            newCurveBSMutate(cc, comm.CLIENT_CONFIG_DELIMITER),
			ExecOptions:       dingoadm.ExecOptions(),
		})
	}

	t.AddStep(&step.SyncFile{ // sync service config
		ContainerSrcId:    &containerId,
		ContainerSrcPath:  fmt.Sprintf("%s/conf/client.conf", root),
		ContainerDestId:   &containerId,
		ContainerDestPath: fmt.Sprintf("%s/conf/client.conf", prefix),
		KVFieldSplit:      comm.CLIENT_CONFIG_DELIMITER,
		Mutate:            newMutate(cc, comm.CLIENT_CONFIG_DELIMITER),
		ExecOptions:       dingoadm.ExecOptions(),
	})
	//t.AddStep(&step.SyncFile{ // sync tools config
	//	ContainerSrcId:    &containerId,
	//	ContainerSrcPath:  fmt.Sprintf("%s/conf/tools.conf", root),
	//	ContainerDestId:   &containerId,
	//	ContainerDestPath: topology.GetCurveFSProjectLayout().ToolsConfSystemPath,
	//	KVFieldSplit:      comm.CLIENT_CONFIG_DELIMITER,
	//	Mutate:            newToolsMutate(cc, comm.CLIENT_CONFIG_DELIMITER),
	//	ExecOptions:       curveadm.ExecOptions(),
	//})
	t.AddStep(&step.TrySyncFile{ // sync tools-v2 config
		ContainerSrcId:    &containerId,
		ContainerSrcPath:  fmt.Sprintf("%s/conf/dingo.yaml", root),
		ContainerDestId:   &containerId,
		ContainerDestPath: topology.GetDingoFSProjectLayout().ToolsV2ConfSystemPath,
		KVFieldSplit:      comm.TOOLS_V2_CONFIG_DELIMITER,
		Mutate:            newToolsV2Mutate(cc, comm.TOOLS_V2_CONFIG_DELIMITER),
		ExecOptions:       dingoadm.ExecOptions(),
	})
	t.AddStep(&step.InstallFile{ // install client.sh shell
		ContainerId:       &containerId,
		ContainerDestPath: createfsScriptPath,
		Content:           &createfsScript,
		ExecOptions:       dingoadm.ExecOptions(),
	})
	t.AddStep(&step.StartContainer{
		ContainerId: &containerId,
		Success:     &success,
		Out:         &out,
		ExecOptions: dingoadm.ExecOptions(),
	})
	t.AddStep(&step.Lambda{
		Lambda: checkStartContainerStatus(&success, &out),
	})
	// TODO(P0): wait mount done

	return t, nil

}

func parseMountPaths(input string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(input, ";")

	for _, pair := range pairs {
		// Split each pair by ':' to separate host and container paths
		paths := strings.Split(pair, ":")
		if len(paths) == 2 {
			hostPath := paths[0]
			containerPath := paths[1]
			result[hostPath] = containerPath
		}
	}

	return result
}
