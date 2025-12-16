/*
 * 	Copyright (c) 2025 dingodb.com Inc.
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
 *
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

package monitor

import (
	"github.com/dingodb/dingoadm/cli/cli"
	comm "github.com/dingodb/dingoadm/internal/common"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/errno"
	"github.com/dingodb/dingoadm/internal/playbook"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	UPGRADE_PLAYBOOK_STEPS = []int{
		playbook.PULL_MONITOR_IMAGE,
		playbook.STOP_MONITOR_SERVICE,
		playbook.CLEAN_MONITOR_SERVICE,
		playbook.CREATE_MONITOR_CONTAINER,
		playbook.START_MONITOR_SERVICE,
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
		Short: "Upgrade monitor service",
		Args:  cliutil.NoArgs,
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
	mcs []*configure.MonitorConfig,
	options upgradeOptions) (*playbook.Playbook, error) {
	steps := UPGRADE_PLAYBOOK_STEPS
	if options.useLocalImage {
		// remove PULL_IMAGE step
		for i, item := range steps {
			if item == playbook.PULL_MONITOR_IMAGE {
				steps = append(steps[:i], steps[i+1:]...)
				break
			}
		}
	}
	pb := playbook.NewPlaybook(dingoadm)
	for _, step := range steps {

		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: mcs,
			Options: map[string]interface{}{
				comm.KEY_CLEAN_ITEMS:      []string{comm.CLEAN_ITEM_CONTAINER},
				comm.KEY_CLEAN_BY_RECYCLE: true,
				comm.KEY_UPGRADE_FLAG:     true,
			},
		})
	}
	return pb, nil
}

func displayTitle(dingoadm *cli.DingoAdm, mcs []*configure.MonitorConfig, options upgradeOptions) {
	total := len(mcs)
	if options.force {
		dingoadm.WriteOutln(color.YellowString("Upgrade %d services at once", total))
	} else {
		dingoadm.WriteOutln(color.YellowString("Upgrade %d services one by one", total))
	}
	dingoadm.WriteOutln(tui.PromptUpgradeService(options.id, options.role, options.host))
}

func upgradeAtOnce(dingoadm *cli.DingoAdm, mcs []*configure.MonitorConfig, options upgradeOptions) error {
	// 1) display upgrade title
	displayTitle(dingoadm, mcs, options)

	// 2) generate upgrade playbook
	pb, err := genUpgradePlaybook(dingoadm, mcs, options)
	if err != nil {
		return err
	}

	// 3) run playbook
	err = pb.Run()
	if err != nil {
		return err
	}

	// 4) print success prompt
	dingoadm.WriteOutln("")
	dingoadm.WriteOutln(color.GreenString("Upgrade %d services success :)", len(mcs)))
	return nil
}

func upgradeOneByOne(dingoadm *cli.DingoAdm, mcs []*configure.MonitorConfig, options upgradeOptions) error {
	// 1) display upgrade title
	displayTitle(dingoadm, mcs, options)

	// 2) upgrade service one by one
	total := len(mcs)
	for i, mc := range mcs {
		// 2.1) confirm by user
		dingoadm.WriteOutln("")
		dingoadm.WriteOutln("Upgrade %s service:", color.BlueString("%d/%d", i+1, total))
		dingoadm.WriteOutln("  + host=%s  role=%s  image=%s", mc.GetHost(), mc.GetRole(), mc.GetImage())
		if pass := tui.ConfirmYes(tui.DEFAULT_CONFIRM_PROMPT); !pass {
			dingoadm.WriteOut(tui.PromptCancelOpetation("upgrade service"))
			return errno.ERR_CANCEL_OPERATION
		}

		// 2.2) generate upgrade playbook
		pb, err := genUpgradePlaybook(dingoadm, []*configure.MonitorConfig{mc}, options)
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
	// 1) parse monitor configure
	mcs, err := configure.ParseMonitor(dingoadm)
	if err != nil {
		return err
	}

	// 2) filter deploy config
	mcs = configure.FilterMonitorConfig(dingoadm, mcs, configure.FilterMonitorOption{
		Id:   options.id,
		Role: options.role,
		Host: options.host,
	})
	if len(mcs) == 0 {
		return errno.ERR_NO_SERVICES_MATCHED
	}

	// 3.1) upgrade service at once
	if options.force {
		return upgradeAtOnce(dingoadm, mcs, options)
	}

	// 3.2) OR upgrade service one by one
	return upgradeOneByOne(dingoadm, mcs, options)
}
