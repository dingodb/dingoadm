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
	"fmt"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

var (
	STOP_PLAYBOOK_STEPS = []int{
		playbook.STOP_SERVICE,
	}
)

type stopOptions struct {
	id    string
	role  string
	host  string
	force bool
}

func NewStopCommand(curveadm *cli.DingoAdm) *cobra.Command {
	var options stopOptions

	cmd := &cobra.Command{
		Use:   "stop [OPTIONS]",
		Short: "Stop service",
		Args:  cliutil.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkCommonOptions(curveadm, options.id, options.role, options.host)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(curveadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.id, "id", "*", "Specify service id")
	flags.StringVar(&options.role, "role", "*", "Specify service role")
	flags.StringVar(&options.host, "host", "*", "Specify service host")
	flags.BoolVarP(&options.force, "force", "f", false, "Never prompt")

	return cmd
}

func genStopPlaybook(curveadm *cli.DingoAdm,
	dcs []*topology.DeployConfig,
	options stopOptions) (*playbook.Playbook, error) {
	dcs = curveadm.FilterDeployConfig(dcs, topology.FilterOption{
		Id:   options.id,
		Role: options.role,
		Host: options.host,
	})
	if len(dcs) == 0 {
		return nil, errno.ERR_NO_SERVICES_MATCHED
	}

	steps := STOP_PLAYBOOK_STEPS
	pb := playbook.NewPlaybook(curveadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: dcs,
		})
	}
	return pb, nil
}

func runStop(curveadm *cli.DingoAdm, options stopOptions) error {
	// 1) parse cluster topology
	dcs, err := curveadm.ParseTopology()
	if err != nil {
		return err
	}

	// 2) generate stop playbook
	pb, err := genStopPlaybook(curveadm, dcs, options)
	if err != nil {
		return err
	}

	// 3) force stop
	if options.force {
		fmt.Print(tui.PromptStopService(options.id, options.role, options.host))
		return pb.Run()
	}

	// 3) confirm by user
	pass := tui.ConfirmYes(tui.PromptStopService(options.id, options.role, options.host))
	if !pass {
		curveadm.WriteOut(tui.PromptCancelOpetation("stop service"))
		return errno.ERR_CANCEL_OPERATION
	}

	// 4) run playground
	return pb.Run()
}
