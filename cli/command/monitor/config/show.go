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
 * Project: dingoadm
 * Author: dongwei (jackblack369)
 */

package config

import (
	"encoding/json"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/internal/configure"
	"github.com/dingodb/dingoadm/internal/errno"
	cliutil "github.com/dingodb/dingoadm/internal/utils"
	"github.com/spf13/cobra"
)

func NewShowCommand(dingoadm *cli.DingoAdm) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "show [OPTIONS]",
		Short: "Show cluster topology",
		Args:  cliutil.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(dingoadm)
		},
		DisableFlagsInUseLine: true,
	}

	return cmd
}

func decodeMonitorJSON(data string) (string, error) {
	monitor := configure.Monitor{}
	err := json.Unmarshal([]byte(data), &monitor)
	if err != nil {
		return "", errno.ERR_DECODE_CLUSTER_POOL_JSON_FAILED.E(err)
	}
	bytes, err := json.MarshalIndent(monitor, "", "    ")
	if err != nil {
		return "", errno.ERR_DECODE_CLUSTER_POOL_JSON_FAILED.E(err)
	}
	return string(bytes), nil
}

func runShow(dingoadm *cli.DingoAdm) error {
	// 1) check whether cluster exist
	if len(dingoadm.Monitor().Monitor) == 0 {
		dingoadm.WriteOutln("<empty monitor>")
		return nil
	}

	//data, err := decodeMonitorJSON(dingoadm.Monitor().Monitor)
	//if err != nil {
	//	return err
	//}
	dingoadm.WriteOutln(dingoadm.Monitor().Monitor)
	return nil
}
