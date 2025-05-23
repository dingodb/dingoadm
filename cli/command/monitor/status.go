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
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	"github.com/dingodb/dingoadm/internal/task/task/monitor"
	tui "github.com/dingodb/dingoadm/internal/tui/service"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

var (
	GET_MONITOR_STATUS_PLAYBOOK_STEPS = []int{
		playbook.INIT_MONITOR_STATUS,
		playbook.GET_MONITOR_STATUS,
	}
)

type statusOptions struct {
	id      string
	role    string
	host    string
	verbose bool
}

func NewStatusCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options statusOptions
	cmd := &cobra.Command{
		Use:   "status [OPTIONS]",
		Short: "Display monitor services status",
		Args:  cliutil.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.id, "id", "*", "Specify monitor service id")
	flags.StringVar(&options.role, "role", "*", "Specify monitor service role")
	flags.StringVar(&options.host, "host", "*", "Specify monitor service host")
	flags.BoolVarP(&options.verbose, "verbose", "v", false, "Verbose output for status")
	return cmd
}

func parseMonitorConfig(dingoadm *cli.DingoAdm) ([]*configure.MonitorConfig, error) {
	if dingoadm.ClusterId() == -1 {
		return nil, errno.ERR_NO_CLUSTER_SPECIFIED
	}
	hosts, hostIps, dcs, err := parseTopology(dingoadm)
	if err != nil {
		return nil, err
	}

	monitor := dingoadm.Monitor()
	return configure.ParseMonitorConfig(dingoadm, "", monitor.Monitor, hosts, hostIps, dcs)
}

func genStatusPlaybook(dingoadm *cli.DingoAdm,
	mcs []*configure.MonitorConfig,
	options statusOptions) (*playbook.Playbook, error) {
	mcs = configure.FilterMonitorConfig(dingoadm, mcs, configure.FilterMonitorOption{
		Id:   options.id,
		Role: options.role,
		Host: options.host,
	})
	if len(mcs) == 0 {
		return nil, errno.ERR_NO_SERVICES_MATCHED
	}

	steps := GET_MONITOR_STATUS_PLAYBOOK_STEPS
	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: mcs,
			ExecOptions: playbook.ExecOptions{
				SilentSubBar:  true,
				SilentMainBar: step == playbook.INIT_MONITOR_STATUS,
				SkipError:     true,
			},
		})
	}
	return pb, nil
}

func displayStatus(dingoadm *cli.DingoAdm, dcs []*configure.MonitorConfig, options statusOptions) {
	statuses := []monitor.MonitorStatus{}
	value := dingoadm.MemStorage().Get(comm.KEY_MONITOR_STATUS)
	if value != nil {
		m := value.(map[string]monitor.MonitorStatus)
		for _, status := range m {
			statuses = append(statuses, status)
		}
	}

	output := tui.FormatMonitorStatus(statuses, options.verbose)
	dingoadm.WriteOutln("")
	dingoadm.WriteOutln("cluster name      : %s", dingoadm.ClusterName())
	dingoadm.WriteOutln("cluster kind      : %s", dcs[0].GetKind())
	dingoadm.WriteOutln("")
	dingoadm.WriteOut("%s", output)
}

func runStatus(dingoadm *cli.DingoAdm, options statusOptions) error {
	// 1) parse monitor config
	mcs, err := parseMonitorConfig(dingoadm)
	if err != nil {
		return err
	}

	// 2) generate get status playbook
	pb, err := genStatusPlaybook(dingoadm, mcs, options)
	if err != nil {
		return err
	}

	// 3) run playground
	err = pb.Run()

	// 4) display service status
	displayStatus(dingoadm, mcs, options)
	return err

}
