/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 * SPDX-License-Identifier: MIT-0
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this
 * software and associated documentation files (the "Software"), to deal in the Software
 * without restriction, including without limitation the rights to use, copy, modify,
 * merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED,
 * INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
 * PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
 * HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
 * OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */
package resource

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	hz       float64
	pageSize int
)

func init() {
	hz = float64(getconf("CLK_TCK"))
	pageSize = getconf("PAGESIZE")
}

func getconf(conf string) int {
	cmd := exec.Command("getconf", conf)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Unable to get system conf '%v', error: %v", conf, err)
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		log.Fatalf("Unable to parse system conf '%v', error: %v, value:'%s'", conf, err, output)
	}
	return value
}

type res struct {
	t time.Time

	utime, stime, cutime, cstime, rss, text, data int
}

type Process struct {
	pid        int
	prev, curr res
}

func FindProcess(pid int) (*Process, error) {
	r, err := getRes(pid)
	if err != nil {
		return nil, err
	}

	return &Process{
		pid:  pid,
		curr: r,
	}, nil
}

func (p *Process) Update() error {
	r, err := getRes(p.pid)
	if err != nil {
		return err
	}
	p.prev = p.curr
	p.curr = r
	return nil
}

func (p *Process) CpuPercent() float64 {
	if p.prev.t.IsZero() || p.curr.t.IsZero() {
		return 0
	}
	dj := float64(p.curr.CpuJiffies() - p.prev.CpuJiffies())
	dt := float64(p.curr.t.Sub(p.prev.t)) / float64(time.Second)
	return dj / (dt * hz) * 100
}

func (p Process) Memory() int {
	return p.curr.Memory()
}

func (p Process) MemoryHuman() string {
	return humanSize(p.Memory())
}

func (p Process) CodeMemory() int {
	return p.curr.MemoryText()
}

func (p Process) DataMemory() int {
	return p.curr.MemoryData()
}

func (p Process) CodeMemoryHuman() string {
	return humanSize(p.curr.MemoryText())
}

func (p Process) DataMemoryHuman() string {
	return humanSize(p.curr.MemoryData())
}

var units = []string{"", "KB", "MB", "GB", "TB", "PB", "EB"}

func humanSize(s int) string {
	var ui = 0
	for s > 10000 && ui < len(units)-1 {
		ui++
		s /= 1024
	}
	return fmt.Sprintf("%d%s", s, units[ui])
}

func (r res) CpuJiffies() int {
	return r.utime + r.stime + r.cutime + r.cstime
}

func (r res) Memory() int {
	return r.rss * pageSize
}

func (r res) MemoryText() int {
	return r.text * pageSize
}

func (r res) MemoryData() int {
	return r.data * pageSize
}

func getRes(pid int) (res, error) {
	r := res{t: time.Now()}

	b, err := ioutil.ReadFile(fmt.Sprintf("/proc/%v/stat", pid))
	if err != nil {
		return r, err
	}
	s := string(b)
	si := strings.Index(s, ")")
	fs := strings.Fields(s[si+1:])

	r.utime, err = strconv.Atoi(fs[11])
	if err != nil {
		return r, err
	}
	r.stime, err = strconv.Atoi(fs[12])
	if err != nil {
		return r, err
	}
	r.cutime, err = strconv.Atoi(fs[13])
	if err != nil {
		return r, err
	}
	r.cstime, err = strconv.Atoi(fs[14])
	if err != nil {
		return r, err
	}

	b, err = ioutil.ReadFile(fmt.Sprintf("/proc/%v/statm", pid))
	if err != nil {
		return r, err
	}
	s = string(b)
	fs = strings.Fields(s)

	r.rss, err = strconv.Atoi(fs[1])
	if err != nil {
		return r, err
	}

	r.text, err = strconv.Atoi(fs[3])
	if err != nil {
		return r, err
	}

	r.data, err = strconv.Atoi(fs[5])
	if err != nil {
		return r, err
	}
	return r, nil
}
