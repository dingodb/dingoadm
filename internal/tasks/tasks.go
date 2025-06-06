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

// __SIGN_BY_WINE93__

package tasks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dingodb/dingoadm/internal/task/task"
	tui "github.com/dingodb/dingoadm/internal/tui/common"
	"github.com/fatih/color"
	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

type (
	ExecOptions struct {
		Concurrency   uint
		SilentMainBar bool
		SilentSubBar  bool
		SkipError     bool
	}

	Tasks struct {
		tasks    []*task.Task
		monitor  *monitor
		wg       sync.WaitGroup
		progress *mpb.Progress
		mainBar  *mpb.Bar
		subBar   map[string]*mpb.Bar
		sync.Mutex
	}
)

func NewTasks() *Tasks {
	wg := sync.WaitGroup{}
	return &Tasks{
		tasks:    []*task.Task{},
		monitor:  newMonitor(),
		wg:       wg,
		progress: mpb.New(mpb.WithWaitGroup(&wg)),
		mainBar:  nil,
		subBar:   map[string]*mpb.Bar{},
	}
}

func (ts *Tasks) AddTask(t ...*task.Task) {
	ts.tasks = append(ts.tasks, t...)
}

func (ts *Tasks) CountPtid(ptid string) int64 {
	var sum int64 = 0
	for _, t := range ts.tasks {
		if t.Ptid() == ptid {
			sum++
		}
	}
	return sum
}

/*
 * before:
 *   host=10.0.0.1 role=mds containerId=1863158e02a6
 *   host=10.0.0.2 role=metaserver containerId=0e6dcd718b85
 *
 * after:
 *   host=10.0.0.1  role=mds         containerId=1863158e02a6
 *   host=10.0.0.2  role=metaserver  containerId=0e6dcd718b85
 */
func (ts *Tasks) prettySubname() {
	lines := [][]interface{}{}
	for _, t := range ts.tasks {
		line := []interface{}{}
		for _, item := range strings.Split(t.Subname(), " ") {
			line = append(line, item)
		}
		lines = append(lines, line)
	}

	output := tui.FixedFormat(lines, 2)
	subnames := strings.Split(output, "\n")
	for i, t := range ts.tasks {
		t.SetSubname(subnames[i])
	}
}

func (ts *Tasks) displayStatus() func(static decor.Statistics) string {
	return func(static decor.Statistics) string {
		if static.Completed {
			status := ts.monitor.get(static.ID)
			if status == STATUS_OK {
				return color.GreenString("[OK]")
			} else if status == STATUS_SKIP {
				return color.YellowString("[SKIP]")
			} else {
				return color.RedString("[ERROR]")
			}
		}
		return ""
	}
}

func (ts *Tasks) displayInstances(t *task.Task) func(static decor.Statistics) string {
	total := ts.CountPtid(t.Ptid())
	return func(static decor.Statistics) string {
		nsucc, nskip, _ := ts.monitor.sum(static.ID)
		return fmt.Sprintf("[%d/%d]", nsucc+nskip, total)
	}
}

func (ts *Tasks) addMainBar() {
	ts.mainBar = ts.progress.Add(1, nil,
		mpb.PrependDecorators(
			decor.Name(ts.tasks[0].Name()+": "),
			decor.OnComplete(decor.Spinner([]string{}), ""),
			decor.Any(ts.displayStatus()),
		),
	)
}

func (ts *Tasks) addSubBar(t *task.Task) {
	ts.Lock()
	defer ts.Unlock()
	if ts.subBar[t.Ptid()] != nil {
		return
	}
	ts.subBar[t.Ptid()] = ts.progress.Add(ts.CountPtid(t.Ptid()), nil,
		mpb.PrependDecorators(
			decor.Name("  + "),
			decor.Name(t.Subname()+" "),
			decor.Any(ts.displayInstances(t), decor.WCSyncWidthR),
			decor.Name(" "),
			decor.OnComplete(decor.Spinner([]string{}), ""),
			decor.Any(ts.displayStatus()),
		),
	)
}

func (ts *Tasks) getSubBar(t *task.Task) *mpb.Bar {
	ts.Lock()
	defer ts.Unlock()
	return ts.subBar[t.Ptid()]
}

func (ts *Tasks) initOptions(options ExecOptions) ExecOptions {
	if options.Concurrency == 0 {
		options.Concurrency = 10
	}
	return options
}

func (ts *Tasks) setMainBarStatus() {
	ts.Lock()
	defer ts.Unlock()
	monitor := ts.monitor
	id := ts.mainBar.ID()
	nsucc := 0
	for _, bar := range ts.subBar {
		status := monitor.get(bar.ID())
		if status == STATUS_ERROR {
			monitor.set(id, monitor.error())
			return
		} else if status == STATUS_OK {
			nsucc++
		}
	}

	if nsucc == 0 { // all task skip
		monitor.set(id, task.ERR_SKIP_TASK)
	} else {
		monitor.set(id, nil)
	}
}

/*
 * Pull Image: [ERROR]
 *   + host=10.0.0.1  image=dingodatabase/dingofs [1/1] [OK]
 *   + host=10.0.0.2  image=dingodatabase/dingofs [1/2] [OK]
 *   + host=10.0.0.3  image=dingodatabase/dingofs [1/10] [ERROR]
 *   + host=10.0.0.1  image=dingodatabase/dingofs [10/10] [OK]
 *   + host=10.0.0.2  image=dingodatabase/dingofs [10/10] [OK]
 *   + host=10.0.0.3  image=dingodatabase/dingofs [1/10] [OK]
 */
func (ts *Tasks) Execute(options ExecOptions) error {
	if len(ts.tasks) == 0 {
		return nil
	}

	ts.prettySubname()
	options = ts.initOptions(options)
	workers := make(chan struct{}, options.Concurrency)
	if !options.SilentMainBar {
		ts.addMainBar()
	}

	// execute task by concurrency
	for _, t := range ts.tasks {
		// FIXME: if we break here, the process bar maybe wait forever
		//        for we didn't execute all the tasks because of the false early appearance
		// if ts.monitor.error() != nil && options.SkipError == false {
		// 	break
		// }
		ts.wg.Add(1)
		workers <- struct{}{}
		if !options.SilentSubBar {
			ts.addSubBar(t)
		}

		// worker
		go func(t *task.Task) {
			bar := ts.getSubBar(t)
			defer func() {
				if bar != nil {
					bar.IncrBy(1)
				}
				<-workers
				ts.wg.Done()
			}()

			// execute task
			id := 0
			if bar != nil {
				id = bar.ID()
			}
			err := t.Execute()
			ts.monitor.set(id, err)
		}(t)
	}

	ts.wg.Wait()
	if ts.mainBar != nil {
		ts.mainBar.IncrBy(1)
		ts.setMainBarStatus()
	}
	ts.progress.Wait()
	return ts.monitor.error()
}
