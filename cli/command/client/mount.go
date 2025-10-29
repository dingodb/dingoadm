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

package client

import (
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	"github.com/dingodb/dingoadm/internal/task/task/fs"
	"github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	MOUNT_EXAMPLE = `Examples:
  $ dingoadm mount fs1  /path/to/mount --host machine -c client.yaml   			   # Mount a classic s3 DingoFS 'fs1' to '/path/to/mount'
  $ dingoadm mount fs2  /path/to/mount --host machine -c client.yaml --new-dingo   # Mount a support rados type DingoFS 'fs2' to '/path/to/mount'
  `
)

var (
	MOUNT_PLAYBOOK_S3_STEPS = []int{
		// TODO(P0): create filesystem
		playbook.CHECK_KERNEL_MODULE,
		playbook.CHECK_CLIENT_S3,
		playbook.MOUNT_FILESYSTEM,
	}

	MOUNT_PLAYBOOK_RADOS_STEPS = []int{
		// TODO(P0): create filesystem
		playbook.CHECK_KERNEL_MODULE,
		playbook.MOUNT_FILESYSTEM,
	}
)

type mountOptions struct {
	host          string
	mountFSName   string
	mountFSType   string
	mountPoint    string
	filename      string
	insecure      bool
	useLocalImage bool
	newDingo      bool // whether to create a new dingo which support rados fs type
}

func checkMountOptions(dingoadm *cli.DingoAdm, options mountOptions) error {
	if !strings.HasPrefix(options.mountPoint, "/") {
		return errno.ERR_FS_MOUNTPOINT_REQUIRE_ABSOLUTE_PATH.
			F("mount point: %s", options.mountPoint)
	}
	return nil
}

func NewMountCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options mountOptions

	cmd := &cobra.Command{
		Use:     "mount NAME_OF_DINGOFS MOUNT_POINT [OPTIONS]",
		Short:   "Mount filesystem",
		Args:    utils.ExactArgs(2),
		Example: MOUNT_EXAMPLE,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			options.mountFSName = args[0]
			options.mountPoint = args[1]
			return checkMountOptions(dingoadm, options)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options.mountFSName = args[0]
			options.mountPoint = args[1]
			return runMount(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.host, "host", "localhost", "Specify target host")
	flags.StringVarP(&options.filename, "conf", "c", "client.yaml", "Specify client configuration file")
	flags.StringVar(&options.mountFSType, "fstype", "vfs_v2", "Specify fs data backend")
	flags.BoolVarP(&options.insecure, "insecure", "k", false, "Mount without precheck")
	flags.BoolVar(&options.useLocalImage, "local", false, "Use local image to mount")
	flags.BoolVar(&options.newDingo, "new-dingo", true, "support create rados type fs")

	return cmd
}

func genMountPlaybook(dingoadm *cli.DingoAdm,
	ccs []*configure.ClientConfig,
	options mountOptions) (*playbook.Playbook, error) {
	steps := MOUNT_PLAYBOOK_S3_STEPS
	if ccs[0].GetStorageType() == configure.STORAGE_TYPE_RADOS {
		steps = MOUNT_PLAYBOOK_RADOS_STEPS
	}
	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {
		if step == playbook.CHECK_KERNEL_MODULE &&
			options.insecure {
			continue
		}

		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: ccs,
			Options: map[string]interface{}{
				comm.KEY_MOUNT_OPTIONS: fs.MountOptions{
					Host:        options.host,
					MountFSName: options.mountFSName,
					MountFSType: options.mountFSType,
					MountPoint:  utils.TrimSuffixRepeat(options.mountPoint, "/"),
				},
				comm.KEY_CLIENT_HOST:              options.host, // for checker
				comm.KEY_CHECK_KERNEL_MODULE_NAME: comm.KERNERL_MODULE_FUSE,
				comm.KEY_USE_LOCAL_IMAGE:          options.useLocalImage,
				comm.KEY_USE_NEW_DINGO:            options.newDingo,
				comm.KEY_FSTYPE:                   options.mountFSType,
			},
			ExecOptions: playbook.ExecOptions{
				SilentSubBar: step == playbook.CHECK_CLIENT_S3,
			},
		})
	}
	return pb, nil
}

func runMount(dingoadm *cli.DingoAdm, options mountOptions) error {
	// 1) parse client configure
	cc, err := configure.ParseClientConfig(options.filename, options.mountFSType)
	if err != nil {
		return err
	} else if cc.GetKind() != topology.KIND_DINGOFS {
		return errno.ERR_REQUIRE_CURVEFS_KIND_CLIENT_CONFIGURE_FILE.
			F("kind: %s", cc.GetKind())
	}

	// 2) generate mount playbook
	pb, err := genMountPlaybook(dingoadm, []*configure.ClientConfig{cc}, options)
	if err != nil {
		return err
	}

	// 3) run playground
	err = pb.Run()
	if err != nil {
		return err
	}

	// 4) print success prompt
	dingoadm.WriteOutln("")
	dingoadm.WriteOutln(color.GreenString("Mount %s to %s (%s) success ^_^"),
		options.mountFSName, options.mountPoint, options.host)
	return nil
}
