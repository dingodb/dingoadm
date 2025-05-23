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
 * Created Date: 2021-12-27
 * Author: Jingli Chen (Wine93)
 */

package command

import (
	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	"github.com/dingodb/dingoadm/internal/task/task/bs"
	tuicomm "github.com/dingodb/dingoadm/internal/tui/common"
	tui "github.com/dingodb/dingoadm/internal/tui/format"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

const (
	FORMAT_EXAMPLE = `Examples:
  $ dingoadm format -f /path/to/format.yaml           # Format chunkfile pool with specified configure file
  $ dingoadm format --status -f /path/to/format.yaml  # Display formatting status
  $ dingoadm format --stop   -f /path/to/format.yaml  # Stop formatting progress`
)

var (
	FORMAT_PLAYBOOK_STEPS = []int{
		playbook.FORMAT_CHUNKFILE_POOL,
	}

	FORMAT_STATUS_PLAYBOOK_STEPS = []int{
		playbook.GET_FORMAT_STATUS,
	}
	// FORMAT_STOP_PLAYBOOK_STEPS stop formatting step
	FORMAT_STOP_PLAYBOOK_STEPS = []int{
		playbook.STOP_FORMAT,
	}
)

type formatOptions struct {
	filename   string
	showStatus bool
	stopFormat bool
	concurrent uint
}

func NewFormatCommand(curveadm *cli.DingoAdm) *cobra.Command {
	var options formatOptions

	cmd := &cobra.Command{
		Use:     "format [OPTIONS]",
		Short:   "Format chunkfile pool",
		Args:    cliutil.NoArgs,
		Example: FORMAT_EXAMPLE,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFormat(curveadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.filename, "formatting", "f", "format.yaml", "Specify the configure file for formatting chunkfile pool")
	flags.BoolVar(&options.showStatus, "status", false, "Show formatting status")
	flags.BoolVar(&options.stopFormat, "stop", false, "Stop formatting progress")
	flags.UintVarP(&options.concurrent, "concurrent", "c", 10, "Specify the number of concurrent for formatting")

	return cmd
}

func genFormatPlaybook(curveadm *cli.DingoAdm,
	fcs []*configure.FormatConfig,
	options formatOptions) (*playbook.Playbook, error) {
	if len(fcs) == 0 {
		return nil, errno.ERR_NO_DISK_FOR_FORMATTING
	}

	if options.showStatus && options.stopFormat {
		return nil, errno.ERR_UNSUPPORT_CONFIGURE_VALUE_TYPE
	}

	steps := FORMAT_PLAYBOOK_STEPS
	if options.showStatus {
		steps = FORMAT_STATUS_PLAYBOOK_STEPS
	}
	if options.stopFormat {
		steps = FORMAT_STOP_PLAYBOOK_STEPS
	}
	pb := playbook.NewPlaybook(curveadm)
	for _, step := range steps {
		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: fcs,
			ExecOptions: playbook.ExecOptions{
				Concurrency:  options.concurrent,
				SilentSubBar: options.showStatus,
			},
		})
	}
	return pb, nil
}

func displayFormatStatus(curveadm *cli.DingoAdm, fcs []*configure.FormatConfig, options formatOptions) {
	statuses := []bs.FormatStatus{}
	v := curveadm.MemStorage().Get(comm.KEY_ALL_FORMAT_STATUS)
	if v != nil {
		m := v.(map[string]bs.FormatStatus)
		for _, status := range m {
			statuses = append(statuses, status)
		}
	}

	output := tui.FormatStatus(statuses)
	curveadm.WriteOutln("")
	curveadm.WriteOut("%s", output)
	return
}

func runFormat(curveadm *cli.DingoAdm, options formatOptions) error {
	// 1) parse format config
	fcs, err := configure.ParseFormat(options.filename)
	if err != nil {
		return err
	}

	// 2) generate start playbook
	pb, err := genFormatPlaybook(curveadm, fcs, options)
	if err != nil {
		return err
	}

	// 3) run playbook
	err = pb.Run()
	if err != nil {
		return err
	}

	// 4) print status or prompt
	if options.showStatus {
		displayFormatStatus(curveadm, fcs, options)
	} else {
		tuicomm.PromptFormat()
	}
	return nil
}
