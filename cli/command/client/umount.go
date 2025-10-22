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
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	"github.com/dingodb/dingoadm/internal/task/task/fs"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	utils "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

var (
	UMOUNT_PLAYBOOK_STEPS = []int{
		playbook.UMOUNT_FILESYSTEM,
	}
)

type umountOptions struct {
	host       string
	mountPoint string
}

func checkUmountOptions(curveadm *cli.DingoAdm, options umountOptions) error {
	if !strings.HasPrefix(options.mountPoint, "/") {
		return errno.ERR_FS_MOUNTPOINT_REQUIRE_ABSOLUTE_PATH.
			F("mount point: %s", options.mountPoint)
	}
	return nil
}

func NewUmountCommand(curveadm *cli.DingoAdm) *cobra.Command {
	var options umountOptions

	cmd := &cobra.Command{
		Use:   "umount MOUNT_POINT [OPTIONS]",
		Short: "Umount filesystem",
		Args:  cliutil.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			options.mountPoint = args[0]
			return checkUmountOptions(curveadm, options)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options.mountPoint = args[0]
			return runUmount(curveadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.host, "host", "localhost", "Specify target host")

	return cmd
}

func genUnmountPlaybook(curveadm *cli.DingoAdm,
	ccs []*configure.ClientConfig,
	options umountOptions) (*playbook.Playbook, error) {
	steps := UMOUNT_PLAYBOOK_STEPS
	pb := playbook.NewPlaybook(curveadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: nil,
			Options: map[string]interface{}{
				comm.KEY_MOUNT_OPTIONS: fs.MountOptions{
					Host:       options.host,
					MountPoint: utils.TrimSuffixRepeat(options.mountPoint, "/"),
				},
			},
		})
	}
	return pb, nil
}

func runUmount(curveadm *cli.DingoAdm, options umountOptions) error {
	// 1) generate unmap playbook
	pb, err := genUnmountPlaybook(curveadm, nil, options)
	if err != nil {
		return err
	}

	// 2) run playground
	return pb.Run()
}
