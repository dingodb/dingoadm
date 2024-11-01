/*
 * Copyright (c) 2024 dingodb.com, Inc. All Rights Reserved
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * Project: DingoFS
 * Created Date: 2024-10-28
 * Author: Wei Dong (jackblack369)
 */

package gateway

import (
	"github.com/dingodb/curveadm/cli/cli"
	comm "github.com/dingodb/curveadm/internal/common"
	"github.com/dingodb/curveadm/internal/configure"
	"github.com/dingodb/curveadm/internal/errno"
	"github.com/dingodb/curveadm/internal/playbook"
	cliutil "github.com/dingodb/curveadm/internal/utils"
	"github.com/spf13/cobra"
)

const (
	START_GATEWAY_EXAMPLE = `Examples:
  $ curveadm gateway start --host dingfs1 --listen-address=:9000 --console-address=:9001 --mountpoint=/home/dingofs/client`
)

type startOptions struct {
	name       string
	host       string
	fileName   string
	mountPoint string
}

var (
	START_GATEWAY_PLAYBOOK_STEPS = []int{
		playbook.START_GATEWAY,
	}
)

func NewStartGatewayCommand(curveadm *cli.CurveAdm) *cobra.Command {
	var options startOptions

	cmd := &cobra.Command{
		// Use:     "start --host={hostName} --listen-address={listenAddr} --console-address={consoleAddr} --mountpoint={path} ",
		Use:     "start {name} {mountpoint} --host dingo7232 -c gateway.yaml ",
		Short:   "start s3 gateway",
		Args:    cliutil.ExactArgs(2),
		Example: START_GATEWAY_EXAMPLE,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.name = args[0]
			options.mountPoint = args[1]
			return runStart(curveadm, options)
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.host, "host", "localhost", "Specify target host")
	flags.StringVarP(&options.fileName, "conf", "c", "gateway.yaml", "Specify gateway configuration file")

	return cmd
}

func runStart(curveadm *cli.CurveAdm, options startOptions) error {

	// 1) generate mount playbook
	pb, err := genStartPlaybook(curveadm, options)
	if err != nil {
		return err
	}

	// 2) run playground
	err = pb.Run()
	if err != nil {
		return err
	}

	// 3) print success prompt
	// curveadm.WriteOutln(color.GreenString("start gateway success !\n gateway listen address: %s%s \n gateway console address: %s%s \n"),
	//	options.host, options.listenAddress, options.host, options.consoleAddress)

	return nil
}

func genStartPlaybook(curveadm *cli.CurveAdm, options startOptions) (*playbook.Playbook, error) {
	steps := START_GATEWAY_PLAYBOOK_STEPS
	pb := playbook.NewPlaybook(curveadm)

	// parse client configure
	gc, err := configure.ParseGatewayConfig(options.fileName)
	if err != nil {
		return nil, err
	}
	listenPort := gc.GetListenPort()
	if listenPort == "" {
		listenPort = "19000"
	}
	gatewayListenAddr := ":" + listenPort

	consolePort := gc.GetConsolePort()
	if consolePort == "" {
		consolePort = "19001"
	}
	gatewayConsoleAddr := ":" + consolePort

	mdsAddr := gc.GetDingofsMDSAddr()
	if mdsAddr == "" {
		return nil, errno.ERR_GATEWAY_MDSADDR_EMPTY
	}

	for _, step := range steps {

		pb.AddStep(&playbook.PlaybookStep{
			Type:    step,
			Configs: gc,
			Options: map[string]interface{}{
				comm.GATEWAY_NAME:         options.name,
				comm.GATEWAY_HOST:         options.host,
				comm.GATEWAY_LISTEN_ADDR:  gatewayListenAddr,
				comm.GATEWAY_CONSOLE_ADDR: gatewayConsoleAddr,
				comm.GATEWAY_MOUNTPOINT:   options.mountPoint,
				comm.MDSADDR:              mdsAddr,
			},
		})
	}
	return pb, nil
}
