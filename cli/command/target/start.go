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
 * Created Date: 2022-02-08
 * Author: Jingli Chen (Wine93)
 */

package target

import (
	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/configure/topology"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	"github.com/dingodb/dingoadm/internal/task/task/bs"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	START_PLAYBOOK_STEPS = []int{
		playbook.START_TARGET_DAEMON,
	}
)

type startOptions struct {
	host     string
	filename string
}

func NewStartCommand(curveadm *cli.DingoAdm) *cobra.Command {
	var options startOptions

	cmd := &cobra.Command{
		Use:   "start [OPTIONS]",
		Short: "Start target deamon",
		Args:  cliutil.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(curveadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.host, "host", "localhost", "Specify target host")
	flags.StringVarP(&options.filename, "conf", "c", "client.yaml", "Specify client configuration file")

	return cmd
}

func genStartPlaybook(curveadm *cli.DingoAdm,
	ccs []*configure.ClientConfig,
	options startOptions) (*playbook.Playbook, error) {
	steps := START_PLAYBOOK_STEPS
	pb := playbook.NewPlaybook(curveadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: ccs,
			Options: map[string]interface{}{
				comm.KEY_TARGET_OPTIONS: bs.TargetOption{
					Host: options.host,
				},
			},
		})
	}
	return pb, nil
}

func runStart(curveadm *cli.DingoAdm, options startOptions) error {
	// 1) parse client configure
	cc, err := configure.ParseClientConfig(options.filename)
	if err != nil {
		return err
	} else if cc.GetKind() != topology.KIND_CURVEBS {
		return errno.ERR_REQUIRE_CURVEBS_KIND_CLIENT_CONFIGURE_FILE.
			F("kind: %s", cc.GetKind())
	}

	// 2) generate map playbook
	pb, err := genStartPlaybook(curveadm, []*configure.ClientConfig{cc}, options)
	if err != nil {
		return err
	}

	// 3) run playground
	err = pb.Run()
	if err != nil {
		return err
	}

	// 4) print success prompt
	curveadm.WriteOutln("")
	curveadm.WriteOutln(color.GreenString("Start target daemon on %s success ^_^"),
		options.host)
	return nil
}
