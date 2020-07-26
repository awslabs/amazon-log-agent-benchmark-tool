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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
	pid, ppid  int
	prev, curr res
	children   []*Process
	prevPs     map[int]*Process
}

func FindProcess(pid int) (*Process, error) {
	ps, err := allProcesses()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve process map: %w", err)
	}

	p, ok := ps[pid]
	if !ok {
		return nil, fmt.Errorf("process with pid %v not found", pid)
	}
	p.prevPs = ps

	return p, nil
}

func (p *Process) Update() error {
	ps, err := allProcesses()
	if err != nil {
		return fmt.Errorf("failed to retrieve process map: %w", err)
	}

	for pid, np := range ps {
		pp, ok := p.prevPs[pid]
		if ok {
			np.prev = pp.curr
		}
	}

	np, ok := ps[p.pid]
	if !ok {
		return fmt.Errorf("process with pid %v no longer exist", p.pid)
	}
	*p = *np
	p.prevPs = ps
	return nil
}

func (p *Process) CpuPercent() float64 {
	if p.prev.t.IsZero() || p.curr.t.IsZero() {
		return 0
	}

	dj := float64(p.allCurrJiffies() - p.allPrevJiffies())
	dt := float64(p.curr.t.Sub(p.prev.t)) / float64(time.Second)
	return dj / (dt * hz) * 100
}

func (p *Process) allCurrJiffies() int {
	j := p.curr.CpuJiffies()
	for _, child := range p.children {
		j += child.allCurrJiffies()
	}
	return j
}

func (p *Process) allPrevJiffies() int {
	j := p.prev.CpuJiffies()
	for _, child := range p.children {
		j += child.allPrevJiffies()
	}
	return j
}

func (p Process) Memory() int {
	m := p.curr.Memory()
	for _, child := range p.children {
		m += child.Memory()
	}
	return m
}

func (p Process) MemoryHuman() string {
	return humanSize(p.Memory())
}

func (p Process) CodeMemory() int {
	m := p.curr.MemoryText()
	for _, child := range p.children {
		m += child.CodeMemory()
	}
	return m
}

func (p Process) DataMemory() int {
	m := p.curr.MemoryData()
	for _, child := range p.children {
		m += child.DataMemory()
	}
	return m
}

func (p Process) CodeMemoryHuman() string {
	return humanSize(p.CodeMemory())
}

func (p Process) DataMemoryHuman() string {
	return humanSize(p.DataMemory())
}

func allPids() ([]int, error) {
	proc, err := os.Open("/proc")
	if err != nil {
		return nil, fmt.Errorf("unable to open '/proc' for reading: %w", err)
	}

	names, err := proc.Readdirnames(0)
	if err != nil {
		return nil, fmt.Errorf("unable to read dirnames from '/proc': %w", err)
	}

	pids := make([]int, 0, len(names))
	for _, n := range names {
		pid, err := strconv.Atoi(n)
		if err == nil {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}

func allProcesses() (map[int]*Process, error) {
	pids, err := allPids()
	if err != nil {
		return nil, err
	}

	ps := make(map[int]*Process)
	for _, pid := range pids {
		p, err := findProcess(pid)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read process %v with error: %w", pid, err)
		}
		ps[pid] = p
	}

	for _, p := range ps {
		pp, ok := ps[p.ppid]
		if !ok {
			continue
		}
		pp.children = append(pp.children, p)
	}
	return ps, nil
}

func findProcess(pid int) (*Process, error) {
	r, ppid, err := readProcess(pid)
	if err != nil {
		return nil, err
	}

	return &Process{
		pid:  pid,
		ppid: ppid,
		curr: r,
	}, nil
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

func readProcess(pid int) (res, int, error) {
	r := res{t: time.Now()}

	b, err := ioutil.ReadFile(fmt.Sprintf("/proc/%v/stat", pid))
	if err != nil {
		return r, 0, err
	}
	s := string(b)
	si := strings.LastIndex(s, ")")
	fs := strings.Fields(s[si+1:])

	// Note: index+3 of fs matches the field number in the kernel doc:
	// https://man7.org/linux/man-pages/man5/proc.5.html
	//fmt.Printf("STAT: 1: '%v', 2: '%v'\n", fs[1], fs[2])
	ppid, err := strconv.Atoi(fs[1])
	if err != nil {
		return r, 0, err
	}

	r.utime, err = strconv.Atoi(fs[11])
	if err != nil {
		return r, ppid, err
	}
	r.stime, err = strconv.Atoi(fs[12])
	if err != nil {
		return r, ppid, err
	}
	r.cutime, err = strconv.Atoi(fs[13])
	if err != nil {
		return r, ppid, err
	}
	r.cstime, err = strconv.Atoi(fs[14])
	if err != nil {
		return r, ppid, err
	}

	b, err = ioutil.ReadFile(fmt.Sprintf("/proc/%v/statm", pid))
	if err != nil {
		return r, ppid, err
	}
	s = string(b)
	fs = strings.Fields(s)

	r.rss, err = strconv.Atoi(fs[1])
	if err != nil {
		return r, ppid, err
	}

	r.text, err = strconv.Atoi(fs[3])
	if err != nil {
		return r, ppid, err
	}

	r.data, err = strconv.Atoi(fs[5])
	if err != nil {
		return r, ppid, err
	}
	return r, ppid, nil
}
