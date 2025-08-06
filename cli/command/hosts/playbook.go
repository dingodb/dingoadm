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
 * Created Date: 2022-10-25
 * Author: Jingli Chen (Wine93)
 */

package hosts

// NOTE: playbook under beta version
import (
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure/hosts"
	"github.com/dingodb/dingoadm/internal/tools"
	"github.com/dingodb/dingoadm/internal/utils"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	retC chan result
	wg   sync.WaitGroup
)

type result struct {
	index int
	host  string
	out   string
	err   error
}

type playbookOptions struct {
	filepath string
	args     []string
	labels   []string
}

func checkPlaybookOptions(dingoadm *cli.DingoAdm, options playbookOptions) error {
	// TODO: added error code
	if !utils.PathExist(options.filepath) {
		return fmt.Errorf("%s: no such file", options.filepath)
	}
	return nil
}

func NewPlaybookCommand(dingoadm *cli.DingoAdm) *cobra.Command {
	var options playbookOptions

	cmd := &cobra.Command{
		Use:   "playbook [OPTIONS] PLAYBOOK [ARGS...]",
		Short: "Execute playbook",
		Args:  cliutil.RequiresMinArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// check args num bigger than 1
			if len(args) == 1 {
				// generate any.sh script to /tmp/any.sh
				anyScript := path.Join("/tmp", "any.sh")
				if !utils.PathExist(anyScript) {
					if err := utils.WriteFile(anyScript, "#!/usr/bin/env bash\n\n\"$@\"\n", 0644); err != nil {
						return fmt.Errorf("write any.sh failed: %w", err)
					}
				}
				args = append([]string{anyScript}, args...) // prepend any.sh to args
			}
			options.filepath = args[0]
			options.args = args[1:]
			return checkPlaybookOptions(dingoadm, options)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlaybook(dingoadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringSliceVarP(&options.labels, "labels", "l", []string{}, "Specify the host labels")

	return cmd
}

func execute(dingoadm *cli.DingoAdm, options playbookOptions, idx int, hc *hosts.HostConfig) {
	defer func() { wg.Done() }()
	name := hc.GetHost()
	target := path.Join("/tmp", utils.RandString(8))
	err := tools.Scp(dingoadm, name, options.filepath, target)
	if err != nil {
		retC <- result{host: name, err: err}
		return
	}

	defer func() {
		command := fmt.Sprintf("rm -rf %s", target)
		tools.ExecuteRemoteCommand(dingoadm, name, command)
	}()

	command := strings.Join([]string{
		strings.Join(hc.GetEnvs(), " "),
		"bash",
		target,
		strings.Join(options.args, " "),
	}, " ")
	out, err := tools.ExecuteRemoteCommand(dingoadm, name, command)
	retC <- result{index: idx, host: name, out: out, err: err}
}

func output(dingoadm *cli.DingoAdm, ret *result) {
	dingoadm.WriteOutln("")
	out, err := ret.out, ret.err
	dingoadm.WriteOutln("%s [%s]", color.YellowString(ret.host),
		utils.Choose(err == nil, color.GreenString("SUCCESS"), color.RedString("FAIL")))
	dingoadm.WriteOutln("---")
	if err != nil {
		dingoadm.Out().Write([]byte(out))
		dingoadm.WriteOutln(err.Error())
	} else if len(out) > 0 {
		dingoadm.Out().Write([]byte(out))
	}
}

func receiver(dingoadm *cli.DingoAdm, total int) {
	dingoadm.WriteOutln("TOTAL: %d hosts", total)
	current := 0
	rets := map[int]result{}
	for ret := range retC {
		rets[ret.index] = ret
		for {
			if v, ok := rets[current]; ok {
				output(dingoadm, &v)
				current++
			} else {
				break
			}
		}
	}
}

func runPlaybook(dingoadm *cli.DingoAdm, options playbookOptions) error {
	var hcs []*hosts.HostConfig
	var err error
	hosts := dingoadm.Hosts()
	if len(hosts) > 0 {
		hcs, err = filter(hosts, options.labels) // filter hosts
		if err != nil {
			return err
		}
	}

	retC = make(chan result)
	wg.Add(len(hcs))
	go receiver(dingoadm, len(hcs))
	for i, hc := range hcs {
		go execute(dingoadm, options, i, hc)
	}
	wg.Wait()
	return nil
}
