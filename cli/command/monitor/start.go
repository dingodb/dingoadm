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
* Created Date: 2023-04-26
* Author: wanghai (SeanHai)
 */

package monitor

import (
	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

var (
	MONITOR_START_STEPS = []int{
		playbook.START_MONITOR_SERVICE,
	}
)

type startOptions struct {
	id   string
	role string
	host string
}

func NewStartCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options startOptions
	cmd := &cobra.Command{
		Use:   "start [OPTIONS]",
		Short: "Start monitor service",
		Args:  cliutil.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.id, "id", "*", "Specify monitor service id")
	flags.StringVar(&options.role, "role", "*", "Specify monitor service role")
	flags.StringVar(&options.host, "host", "*", "Specify monitor service host")

	return cmd
}

func genStartPlaybook(dingoadm *cli.DingoAdm,
	mcs []*configure.MonitorConfig,
	options startOptions) (*playbook.Playbook, error) {
	mcs = configure.FilterMonitorConfig(dingoadm, mcs, configure.FilterMonitorOption{
		Id:   options.id,
		Role: options.role,
		Host: options.host,
	})
	if len(mcs) == 0 {
		return nil, errno.ERR_NO_SERVICES_MATCHED
	}

	steps := MONITOR_START_STEPS
	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: mcs,
		})
	}
	return pb, nil
}

func runStart(dingoadm *cli.DingoAdm, options startOptions) error {
	// 1) parse monitor configure
	mcs, err := parseMonitorConfig(dingoadm)
	if err != nil {
		return err
	}

	// 2) generate start playbook
	pb, err := genStartPlaybook(dingoadm, mcs, options)
	if err != nil {
		return err
	}

	// 3) confirm by user
	if pass := tui.ConfirmYes(tui.PromptStartService(options.id, options.role, options.host)); !pass {
		dingoadm.WriteOut(tui.PromptCancelOpetation("start monitor service"))
		return errno.ERR_CANCEL_OPERATION
	}

	// 4) run playground
	return pb.Run()
}
