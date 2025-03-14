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
 * Created Date: 2022-02-09
 * Author: Jingli Chen (Wine93)
 */

package target

import (
	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/playbook"
	"github.com/dingodb/dingoadm/internal/task/task/bs"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	DELETE_PLAYBOOK_STEPS = []int{
		playbook.DELETE_TARGET,
	}
)

type deleteOptions struct {
	host string
	tid  string
}

func NewDeleteCommand(curveadm *cli.DingoAdm) *cobra.Command {
	var options deleteOptions

	cmd := &cobra.Command{
		Use:     "rm TID [OPTIONS]",
		Aliases: []string{"delete"},
		Short:   "Delete a target of CurveBS",
		Args:    cliutil.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.tid = args[0]
			return runDelete(curveadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.host, "host", "localhost", "Specify target host")

	return cmd
}

func genDeletePlaybook(curveadm *cli.DingoAdm, options deleteOptions) (*playbook.Playbook, error) {
	steps := DELETE_PLAYBOOK_STEPS
	pb := playbook.NewPlaybook(curveadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: nil,
			Options: map[string]interface{}{
				comm.KEY_TARGET_OPTIONS: bs.TargetOption{
					Host: options.host,
					Tid:  options.tid,
				},
			},
		})
	}
	return pb, nil
}

func runDelete(curveadm *cli.DingoAdm, options deleteOptions) error {
	// 1) generate list playbook
	pb, err := genDeletePlaybook(curveadm, options)
	if err != nil {
		return err
	}

	// 2) run playground
	err = pb.Run()
	if err != nil {
		return err
	}

	// 3) print targets
	curveadm.WriteOutln("")
	curveadm.WriteOutln(color.GreenString("Delete target (tid=%s) on %s success ^_^"),
		options.tid, options.host)
	return nil
}
