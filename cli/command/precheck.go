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
 * Created Date: 2022-07-11
 * Author: Jingli Chen (Wine93)
 */

package command

import (
	"fmt"
	"strings"

	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	utils "github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	PRECHECK_EXAMPLE = `Examples:
  $ dingoadm precheck                         # Check all items
  $ dingoadm precheck --skip topology         # Check all items except topology
  $ dingoadm precheck --skip topology,kernel  # Check all items except topology and kernel`
)

const (
	CHECK_ITEM_TOPOLOGY   = "topology"
	CHECK_ITEM_SSH        = "ssh"
	CHECK_ITEM_PERMISSION = "permission"
	CHECK_ITEM_KERNEL     = "kernel"
	CHECK_ITEM_NERWORK    = "network"
	CHECK_ITEM_DATE       = "date"
	CHECK_ITEM_SERVICE    = "service"
)

var (
	CURVEBS_PRECHECK_STEPS = []int{
		playbook.CHECK_TOPOLOGY,             // topology
		playbook.CHECK_SSH_CONNECT,          // ssh
		playbook.CHECK_PERMISSION,           // permission
		playbook.CHECK_KERNEL_VERSION,       // kernel
		playbook.CLEAN_PRECHECK_ENVIRONMENT, // <none>
		playbook.CHECK_PORT_IN_USE,          // network
		playbook.CHECK_DESTINATION_REACHABLE,
		playbook.START_HTTP_SERVER,
		playbook.CHECK_NETWORK_FIREWALL,
		playbook.GET_HOST_DATE, // date
		playbook.CHECK_HOST_DATE,
		playbook.CHECK_CHUNKFILE_POOL, // service
		//playbook.CHECK_S3,
	}

	DINGOFS_PRECHECK_STEPS = []int{
		playbook.CHECK_TOPOLOGY,             // topology
		playbook.CHECK_SSH_CONNECT,          // ssh
		playbook.CHECK_PERMISSION,           // permission
		playbook.CLEAN_PRECHECK_ENVIRONMENT, // <none>
		playbook.CHECK_PORT_IN_USE,          // network
		playbook.START_HTTP_SERVER,
		playbook.CHECK_DESTINATION_REACHABLE,
		// playbook.CHECK_NETWORK_FIREWALL,
		playbook.GET_HOST_DATE, // date
		playbook.CHECK_HOST_DATE,
	}

	PRECHECK_POST_STEPS = []int{
		playbook.CLEAN_PRECHECK_ENVIRONMENT,
	}

	BELONG_CHECK_ITEM = map[int]string{
		playbook.CHECK_TOPOLOGY:              CHECK_ITEM_TOPOLOGY,
		playbook.CHECK_SSH_CONNECT:           CHECK_ITEM_SSH,
		playbook.CHECK_PERMISSION:            CHECK_ITEM_PERMISSION,
		playbook.CHECK_KERNEL_VERSION:        CHECK_ITEM_KERNEL,
		playbook.CHECK_PORT_IN_USE:           CHECK_ITEM_NERWORK,
		playbook.CHECK_DESTINATION_REACHABLE: CHECK_ITEM_NERWORK,
		playbook.CHECK_NETWORK_FIREWALL:      CHECK_ITEM_NERWORK,
		playbook.GET_HOST_DATE:               CHECK_ITEM_DATE,
		playbook.CHECK_HOST_DATE:             CHECK_ITEM_DATE,
		playbook.CHECK_CHUNKFILE_POOL:        CHECK_ITEM_SERVICE,
		playbook.CHECK_S3:                    CHECK_ITEM_SERVICE,
	}

	CHECK_ITEMS = []string{
		CHECK_ITEM_TOPOLOGY,
		CHECK_ITEM_SSH,
		CHECK_ITEM_PERMISSION,
		CHECK_ITEM_KERNEL,
		CHECK_ITEM_NERWORK,
		CHECK_ITEM_DATE,
		CHECK_ITEM_SERVICE,
	}
)

type precheckOptions struct {
	skipSnapshotClone bool
	skip              []string
	useLocalImage     bool
	//only              []string
}

func checkPrecheckOptions(options precheckOptions) error {
	supported := utils.Slice2Map(CHECK_ITEMS)
	for _, role := range options.skip {
		if !supported[role] {
			return errno.ERR_UNSUPPORT_SKIPPED_CHECK_ITEM
		}
	}
	return nil
}

func NewPrecheckCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options precheckOptions

	cmd := &cobra.Command{
		Use:     "precheck",
		Short:   "Precheck environment",
		Args:    cliutil.NoArgs,
		Example: PRECHECK_EXAMPLE,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkPrecheckOptions(options)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrecheck(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	usage := fmt.Sprintf("Specify skipped check item (%s)", strings.Join(CHECK_ITEMS, ","))
	flags.StringSliceVar(&options.skip, "skip", []string{}, usage)
	flags.BoolVar(&options.useLocalImage, "local", false, "Use local image")
	//flags.StringSliceVar(&options.only, "only", CHECK_ITEMS, usage)

	return cmd
}

func skipPrecheckSteps(precheckSteps []int, options precheckOptions) []int {
	out := []int{}
	skipped := utils.Slice2Map(options.skip)

	if options.useLocalImage {
		// remove PULL_IMAGE step
		for i, item := range precheckSteps {
			if item == PULL_IMAGE {
				precheckSteps = append(precheckSteps[:i], precheckSteps[i+1:]...)
				break
			}
		}
	}

	for _, step := range precheckSteps {
		if skipped[BELONG_CHECK_ITEM[step]] {
			continue
		}
		out = append(out, step)
	}
	return out
}

func genPrecheckPlaybook(dingoadm *cli.DingoAdm,
	dcs []*topology.DeployConfig,
	options precheckOptions) (*playbook.Playbook, error) {
	kind := dcs[0].GetKind()
	steps := DINGOFS_PRECHECK_STEPS
	if kind == topology.KIND_CURVEBS {
		steps = CURVEBS_PRECHECK_STEPS
	}

	allowAbsent := false
	roles := dingoadm.GetRoles(dcs)
	if kind == topology.KIND_DINGOFS && !utils.Contains(roles, topology.ROLE_METASERVER) {
		// if no metaservers, allow absent metaservers
		allowAbsent = true
	}

	steps = skipPrecheckSteps(steps, options)

	// add playbook step
	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {
		configs := dcs
		switch step {
		case playbook.CHECK_TOPOLOGY:
			configs = configs[:1] // any deploy config
		case playbook.CHECK_KERNEL_VERSION:
			// TODO:
			configs = dingoadm.FilterDeployConfigByRole(dcs, ROLE_CHUNKSERVER)
		case playbook.CHECK_HOST_DATE:
			configs = configs[:1]
		case playbook.CHECK_CHUNKFILE_POOL:
			configs = dingoadm.FilterDeployConfigByRole(dcs, ROLE_CHUNKSERVER)
		}

		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: configs,
			Options: map[string]interface{}{
				comm.KEY_ALL_DEPLOY_CONFIGS:       dcs,
				comm.KEY_CHECK_WITH_WEAK:          false,
				comm.KEY_CHECK_SKIP_SNAPSHOECLONE: options.skipSnapshotClone,
				comm.KEY_ALLOW_ABSENT:             allowAbsent,
			},
			ExecOptions: playbook.ExecOptions{
				SilentSubBar: step == playbook.CHECK_HOST_DATE,
			},
		})
	}

	// add playbook post steps
	steps = PRECHECK_POST_STEPS
	for _, step := range steps {
		pb.AddPostStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: dcs,
			ExecOptions: playbook.ExecOptions{
				SilentSubBar: true,
			},
		})
	}

	return pb, nil
}

func runPrecheck(dingoadm *cli.DingoAdm, options precheckOptions) error {
	// 1) parse cluster topology
	dcs, err := dingoadm.ParseTopology()
	if err != nil {
		return err
	}

	// 2) generate precheck playbook
	pb, err := genPrecheckPlaybook(dingoadm, dcs, options)
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
	dingoadm.WriteOutln(color.GreenString("Congratulations!!! all precheck passed :)"))
	return nil
}
