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
 * Created Date: 2022-01-16
 * Author: Jingli Chen (Wine93)
 */

// __SIGN_BY_WINE93__

package command

import (
	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/internal/utils"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	UPGRADE_PLAYBOOK_STEPS = []int{
		// TODO(P0): we can skip it for upgrade one service more than once
		playbook.PULL_IMAGE,
		playbook.STOP_SERVICE,
		playbook.CLEAN_SERVICE,
		playbook.CREATE_CONTAINER,
		playbook.SYNC_CONFIG,
		playbook.START_SERVICE,
	}

	UPGRADE_STORE_MDSV2_STEPS = []int{
		playbook.PULL_IMAGE,
		playbook.STOP_SERVICE,
		playbook.CLEAN_SERVICE,
		playbook.CREATE_CONTAINER,
		playbook.SYNC_CONFIG,
		playbook.START_COORDINATOR,
		playbook.START_STORE,
		playbook.CHECK_STORE_HEALTH,
		playbook.START_MDS_V2,
	}
)

type upgradeOptions struct {
	id            string
	role          string
	host          string
	force         bool
	useLocalImage bool
}

func NewUpgradeCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options upgradeOptions

	cmd := &cobra.Command{
		Use:   "upgrade [OPTIONS]",
		Short: "Upgrade service",
		Args:  cliutil.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkCommonOptions(dingoadm, options.id, options.role, options.host)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.id, "id", "*", "Specify service id")
	flags.StringVar(&options.role, "role", "*", "Specify service role")
	flags.StringVar(&options.host, "host", "*", "Specify service host")
	flags.BoolVarP(&options.force, "force", "f", false, "Never prompt")
	flags.BoolVar(&options.useLocalImage, "local", false, "Use local image")

	return cmd
}

func genUpgradePlaybook(dingoadm *cli.DingoAdm,
	dcs []*topology.DeployConfig,
	options upgradeOptions) (*playbook.Playbook, error) {
	dcs = dingoadm.FilterDeployConfig(dcs, topology.FilterOption{
		Id:   options.id,
		Role: options.role,
		Host: options.host,
	})
	if len(dcs) == 0 {
		return nil, errno.ERR_NO_SERVICES_MATCHED
	}
	steps := UPGRADE_PLAYBOOK_STEPS
	roles := dingoadm.GetRoles(dcs)
	if utils.Contains(roles, topology.ROLE_MDS_V2) {
		// upgrade mds v2
		steps = UPGRADE_STORE_MDSV2_STEPS
	}

	if options.useLocalImage {
		// remove PULL_IMAGE step
		for i, item := range steps {
			if item == PULL_IMAGE {
				steps = append(steps[:i], steps[i+1:]...)
				break
			}
		}
	}
	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {

		// fliter deploy config according filte rule
		stepDcs := dcs
		if len(DEPLOY_FILTER_ROLE[step]) > 0 {
			role := DEPLOY_FILTER_ROLE[step]
			stepDcs = dingoadm.FilterDeployConfigByRole(stepDcs, role)
		}

		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: stepDcs,
			Options: map[string]interface{}{
				comm.KEY_CLEAN_ITEMS:      []string{comm.CLEAN_ITEM_CONTAINER},
				comm.KEY_CLEAN_BY_RECYCLE: true,
				comm.KEY_SKIP_MDSV2_CLI:   true,
			},
		})
	}
	return pb, nil
}

func displayTitle(dingoadm *cli.DingoAdm, dcs []*topology.DeployConfig, options upgradeOptions) {
	total := len(dcs)
	if options.force {
		dingoadm.WriteOutln(color.YellowString("Upgrade %d services at once", total))
	} else {
		dingoadm.WriteOutln(color.YellowString("Upgrade %d services one by one", total))
	}
	dingoadm.WriteOutln(color.YellowString("Upgrade services: %s", serviceStats(dcs)))
}

func upgradeAtOnce(dingoadm *cli.DingoAdm, dcs []*topology.DeployConfig, options upgradeOptions) error {
	// 1) display upgrade title
	displayTitle(dingoadm, dcs, options)

	// 2) confirm by user
	// if pass := tui.ConfirmYes(tui.DEFAULT_CONFIRM_PROMPT); !pass {
	// 	dingoadm.WriteOut(tui.PromptCancelOpetation("upgrade service"))
	// 	return errno.ERR_CANCEL_OPERATION
	// }

	// 3) generate upgrade playbook
	pb, err := genUpgradePlaybook(dingoadm, dcs, options)
	if err != nil {
		return err
	}

	// 4) run playbook
	err = pb.Run()
	if err != nil {
		return err
	}

	// 5) print success prompt
	dingoadm.WriteOutln("")
	dingoadm.WriteOutln(color.GreenString("Upgrade %d services success :)", len(dcs)))
	return nil
}

func upgradeOneByOne(dingoadm *cli.DingoAdm, dcs []*topology.DeployConfig, options upgradeOptions) error {
	// 1) display upgrade title
	displayTitle(dingoadm, dcs, options)

	// 2) upgrade service one by one
	total := len(dcs)
	for i, dc := range dcs {
		// 2.1) confirm by user
		dingoadm.WriteOutln("")
		dingoadm.WriteOutln("Upgrade %s service:", color.BlueString("%d/%d", i+1, total))
		dingoadm.WriteOutln("  + host=%s  role=%s  image=%s", dc.GetHost(), dc.GetRole(), dc.GetContainerImage())
		if pass := tui.ConfirmYes(tui.DEFAULT_CONFIRM_PROMPT); !pass {
			dingoadm.WriteOut(tui.PromptCancelOpetation("upgrade service"))
			return errno.ERR_CANCEL_OPERATION
		}

		// 2.2) generate upgrade playbook
		pb, err := genUpgradePlaybook(dingoadm, []*topology.DeployConfig{dc}, options)
		if err != nil {
			return err
		}

		// 2.3) run playbook
		err = pb.Run()
		if err != nil {
			return err
		}

		// 2.4) print success prompt
		dingoadm.WriteOutln("")
		dingoadm.WriteOutln(color.GreenString("Upgrade %d/%d sucess :)"), i+1, total)
	}
	return nil
}

func runUpgrade(dingoadm *cli.DingoAdm, options upgradeOptions) error {
	// 1) parse cluster topology
	dcs, err := dingoadm.ParseTopology()
	if err != nil {
		return err
	}

	// 2) filter deploy config
	dcs = dingoadm.FilterDeployConfig(dcs, topology.FilterOption{
		Id:   options.id,
		Role: options.role,
		Host: options.host,
	})
	if len(dcs) == 0 {
		return errno.ERR_NO_SERVICES_MATCHED
	}

	// 3.1) upgrade service at once
	if options.force {
		return upgradeAtOnce(dingoadm, dcs, options)
	}

	// 3.2) OR upgrade service one by one
	return upgradeOneByOne(dingoadm, dcs, options)
}
