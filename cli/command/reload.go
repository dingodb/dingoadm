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
 */

// __SIGN_BY_WINE93__

package command

import (
	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	RELOAD_PLAYBOOK_STEPS = []int{
		playbook.SYNC_CONFIG,
		playbook.RESTART_SERVICE,
	}
)

type reloadOptions struct {
	id   string
	role string
	host string
}

func NewReloadCommand(curveadm *cli.DingoAdm) *cobra.Command {
	var options reloadOptions

	cmd := &cobra.Command{
		Use:   "reload [OPTIONS]",
		Short: "Reload service",
		Args:  cliutil.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkCommonOptions(curveadm, options.id, options.role, options.host)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReload(curveadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.id, "id", "*", "Specify service id")
	flags.StringVar(&options.role, "role", "*", "Specify service role")
	flags.StringVar(&options.host, "host", "*", "Specify service host")

	return cmd
}

func genReloadPlaybook(curveadm *cli.DingoAdm,
	dcs []*topology.DeployConfig,
	options reloadOptions) (*playbook.Playbook, error) {
	dcs = curveadm.FilterDeployConfig(dcs, topology.FilterOption{
		Id:   options.id,
		Role: options.role,
		Host: options.host,
	})
	if len(dcs) == 0 {
		return nil, errno.ERR_NO_SERVICES_MATCHED
	}

	steps := RELOAD_PLAYBOOK_STEPS
	pb := playbook.NewPlaybook(curveadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: dcs,
		})
	}
	return pb, nil
}

func runReload(curveadm *cli.DingoAdm, options reloadOptions) error {
	// 1) parse cluster topology
	dcs, err := curveadm.ParseTopology()
	if err != nil {
		return err
	}

	// 2) generate restart playbook
	pb, err := genReloadPlaybook(curveadm, dcs, options)
	if err != nil {
		return err
	}

	// 3) confirm by user
	if pass := tui.ConfirmYes(tui.PromptReloadService(options.id, options.role, options.host)); !pass {
		curveadm.WriteOut(tui.PromptCancelOpetation("reload service"))
		return errno.ERR_CANCEL_OPERATION
	}

	// 4) run playground
	err = pb.Run()
	if err != nil {
		return err
	}

	// 5) print success prompt
	curveadm.WriteOutln("")
	curveadm.WriteOutln(color.GreenString("Reload success :)"))
	return nil
}
