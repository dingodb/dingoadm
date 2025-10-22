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
* Created Date: 2023-04-27
* Author: wanghai (SeanHai)
*
* Project: Dingoadm
* Author: jackblack369 (Dongwei)
 */

package monitor

import (
	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/dingodb/dingoadm/internal/utils"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

const (
	CLEAN_EXAMPLE = `Examples:
  $ dingoadm monitor clean                                  # Clean everything for monitor
  $ dingoadm monitor clean --only='data'                    # Clean data for monitor
  $ dingoadm monitor clean --role=grafana --only=container  # Clean container for grafana service`
)

var (
	CLEAN_PLAYBOOK_STEPS = []int{
		playbook.CLEAN_MONITOR,
	}

	CLEAN_ITEMS = []string{
		comm.CLEAN_ITEM_DATA,
		comm.CLEAN_ITEM_CONTAINER,
	}
)

type cleanOptions struct {
	id    string
	role  string
	host  string
	only  []string
	force bool
}

func NewCleanCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options cleanOptions

	cmd := &cobra.Command{
		Use:     "clean [OPTIONS]",
		Short:   "Clean monitor's environment",
		Args:    cliutil.NoArgs,
		Example: CLEAN_EXAMPLE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClean(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.id, "id", "*", "Specify monitor service id")
	flags.StringVar(&options.role, "role", "*", "Specify monitor service role")
	flags.StringVar(&options.host, "host", "*", "Specify monitor service host")
	flags.StringSliceVarP(&options.only, "only", "o", CLEAN_ITEMS, "Specify clean item")
	flags.BoolVarP(&options.force, "force", "f", false, "Force to clean without confirmation")
	return cmd
}

func genCleanPlaybook(dingoadm *cli.DingoAdm,
	mcs []*configure.MonitorConfig,
	options cleanOptions) (*playbook.Playbook, error) {
	mcs = configure.FilterMonitorConfig(dingoadm, mcs, configure.FilterMonitorOption{
		Id:   options.id,
		Role: options.role,
		Host: options.host,
	})
	if len(mcs) == 0 {
		return nil, errno.ERR_NO_SERVICES_MATCHED
	}
	steps := CLEAN_PLAYBOOK_STEPS
	// check if options's only item include container
	if utils.Contains(options.only, comm.CLEAN_ITEM_CONTAINER) {
		// add stop service step before clean service step
		steps = append([]int{playbook.STOP_MONITOR_SERVICE}, steps...)
	}
	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: mcs,
			Options: map[string]interface{}{
				comm.KEY_CLEAN_ITEMS: options.only,
			},
		})
	}
	return pb, nil
}

func runClean(dingoadm *cli.DingoAdm, options cleanOptions) error {
	// 1) parse monitor config
	mcs, err := parseMonitorConfig(dingoadm)
	if err != nil {
		return err
	}

	// 2) generate clean playbook
	pb, err := genCleanPlaybook(dingoadm, mcs, options)
	if err != nil {
		return err
	}

	if !options.force {
		// 3) confirm by user
		if pass := tui.ConfirmYes(tui.PromptCleanService(options.role, options.host, options.only)); !pass {
			dingoadm.WriteOut(tui.PromptCancelOpetation("clean monitor service"))
			return errno.ERR_CANCEL_OPERATION
		}
	}

	// 4) run playground
	err = pb.Run()
	if err != nil {
		return err
	}
	return nil
}
