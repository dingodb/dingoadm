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
 * Created Date: 2021-10-15
 * Author: Jingli Chen (Wine93)
 */

package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/dingodb/dingoadm/cli/cli"
	"github.com/dingodb/dingoadm/cli/command"
)

func Execute() {
	dingoadm, err := cli.NewDingoAdm()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	id := dingoadm.PreAudit(time.Now(), os.Args[1:])
	cmd := command.NewDingoAdmCommand(dingoadm)
	err = cmd.Execute()
	dingoadm.PostAudit(id, err)
	if err != nil {
		os.Exit(1)
	}
}
