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
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/awslabs/amazon-log-agent-benchmark-tool/generator"
	"github.com/awslabs/amazon-log-agent-benchmark-tool/resource"
	"github.com/awslabs/amazon-log-agent-benchmark-tool/rotator"
)

const (
	FixedLogLine = "INFO CloudWatchOutput      Amazon::Monitoring::CloudWatchOutput::new - CloudWatchOutput sender=data/cloudwatch/current endpoint=https://monitoring.us-east-1.amazonaws.com maxBytes=76800"
	noPid        = -1
)

type MultpleValueFlag []string

func (f *MultpleValueFlag) String() string {
	return fmt.Sprintf("%v", []string(*f))
}

func (f *MultpleValueFlag) Set(value string) error {
	values := strings.Split(value, ",")
	*f = append(*f, values...)
	return nil
}

var Usage = func() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n%v -log LOGFILE1,LOGFILE2 -log LOGFILE3 COMMAND PARAM1 PARAM2\n\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var logfiles, rateStrs MultpleValueFlag
	var tLength, rampUp, freq, rotateDuration time.Duration
	var timeLayout, logLine, rotateSizeStr string
	var pid, rotateKeep int
	var pipeOutput bool
	flag.Var(&logfiles, "log", "Path of the log files being generated and writes logs to, you can specify multiple values by using the parameter multiple times or use comma seperated list")
	flag.Var(&rateStrs, "rate", "Log generation rate to be tested, e.g. -log 1,100,1k,10k,100k, default 100")
	flag.IntVar(&pid, "p", noPid, "Pid of the agent to check resource usage")
	flag.BoolVar(&pipeOutput, "o", false, "Pipe agent output to stdout and stderr")
	flag.StringVar(&timeLayout, "timelayout", "Jan _2 15:04:05.000000000", "Format to print the timestamp for the log lines, following Go time layout, see: https://golang.org/pkg/time/#pkg-constants")
	flag.StringVar(&logLine, "line", FixedLogLine, "Content of the log line to be used")
	flag.DurationVar(&tLength, "t", 10*time.Second, "Test duration, in format supported by time.ParseDuration, default 10s")
	flag.DurationVar(&rampUp, "r", 1*time.Second, "Ramp up duration, time for agent to stablize, stats will not be collected during the ramp up, default 1s")
	flag.DurationVar(&freq, "f", 1*time.Second, "Frequency to collect metrics represented in time duration, default 1s")

	flag.IntVar(&rotateKeep, "rotatekeep", 0, "Number of rotation files to keep, 0 to disable rotation")
	flag.StringVar(&rotateSizeStr, "rotatesize", "", "Size of the logfile before rotation")
	flag.DurationVar(&rotateDuration, "rotatetime", 0, "How much time the logfile should be rotated")

	flag.Parse()

	if len(logfiles) == 0 {
		log.Printf("Expecting at least one log parameter to write logs to, none given")
		Usage()
		os.Exit(1)
	}

	rates, err := parseRates(rateStrs)
	if err != nil {
		log.Printf("Unable to parse rate param: %v", err)
		Usage()
		os.Exit(1)
	}
	if len(rates) == 0 {
		rates = []float64{100}
	}

	rsize, err := parseNumber(rotateSizeStr)
	if err != nil {
		log.Printf("Unable to parse rate param: %v", err)
		Usage()
		os.Exit(1)
	}
	rconf := rotator.Config{
		Keep:     rotateKeep,
		Duration: rotateDuration,
		Size:     int64(rsize),
	}

	var opts []generator.Opt
	if timeLayout != "" {
		opts = append(opts, generator.OptTimeLayout(timeLayout))
	}

	files, err := createLogFiles(logfiles, rconf)
	if err != nil {
		log.Fatalf("Failed to create logfiles: %v", err)
	}

	gens := generator.NewFixed(logLine, files, opts...)

	args := flag.Args()
	var cmd *exec.Cmd
	if len(args) > 0 {
		cmd, err = startAgent(pipeOutput, args[0], args[1:])
		if err != nil {
			log.Fatalf("Failed to start agent with error: %v", err)
		}
		pid = cmd.Process.Pid
		fmt.Println("Agent running with PID: ", pid)
	}

	if pid == noPid {
		fmt.Println("No agent command or agent pid given, just generating logs instead.")
	}

	for _, rate := range rates {
		gens.SetRate(rate)
		fmt.Printf("Ramping up for rate %v for %v ...\n", rate, rampUp)
		time.Sleep(rampUp)
		start := time.Now()
		t := time.NewTicker(freq)

		var scpu, sres, mres float64
		var n int
		var p *resource.Process
		if pid != noPid {
			p, err = resource.FindProcess(pid)
			if err != nil {
				if len(args) > 0 {
					log.Fatalf("Failed to find process for command %v with params %v, pid: %v, error: %v", args[0], args[1:], pid, err)
				} else {
					log.Fatalf("Failed to find process for pid: %v, error: %v", pid, err)
				}
			}
			// Initialize cpu usage data
			err = p.Update()
			if err != nil {
				log.Fatalf("Failed to initialize resource usage: %v", err)
			}
			<-t.C
		}

		for n = 0; time.Now().Sub(start) < tLength; n++ {
			if p != nil {
				err = p.Update()
				if err != nil {
					log.Fatalf("Failed to update resource usage: %v", err)
				}
				cpu := p.CpuPercent()
				fmt.Printf("CPU: %.1f%% MEM: %v \n", cpu, p.MemoryHuman())
				scpu += cpu
				mbf := float64(p.Memory())
				sres += mbf
				if mres < mbf {
					mres = mbf
				}
			} else {
				fmt.Printf(".")
			}
			<-t.C
		}
		fmt.Println()
		t.Stop()
		if pid != noPid && n > 0 {
			fmt.Printf("In the past %v, average cpu usage: %.1f%%, average memory usage: %.1fM, maximium memory usage: %.1fM\n\n", tLength, scpu/float64(n), sres/float64(n)/1024/1024, mres/1024/1024)
		}
	}

	fmt.Println("Stopping generators ...")
	gens.Stop()
	if len(args) > 0 {
		fmt.Println("Stopping the agent ...")
		stopAgent(cmd)
	}
}

func createLogFiles(paths []string, rconf rotator.Config) ([]io.Writer, error) {
	var ws []io.Writer
	for _, path := range paths {
		w, err := rotator.NewWriter(rotator.NewFileRotator(path, rconf.Keep), rconf)
		if err != nil {
			return nil, fmt.Errorf("failed to create file %v: %w", path, err)
		}
		ws = append(ws, w)
	}
	return ws, nil
}

func startAgent(pipeOutput bool, c string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(c, args...)
	if pipeOutput {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		go func() {
			io.Copy(os.Stdout, stdout)
		}()
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, err
		}
		go func() {
			io.Copy(os.Stderr, stderr)
		}()
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to execute command %v with params %v, error: %v", args[0], args[1:], err)
	}
	return cmd, nil
}

func stopAgent(cmd *exec.Cmd) {
	cmd.Process.Signal(syscall.SIGINT)
	go func() {
		time.Sleep(5 * time.Second)
		log.Println("Agent still alive 5 seconds after SIGINT, kill now")
		cmd.Process.Kill()
	}()
	err := cmd.Wait()
	log.Printf("Agent exited state: %v, error: %v", cmd.ProcessState, err)
}

func parseRates(strs []string) ([]float64, error) {
	var result []float64
	for _, str := range strs {
		n, err := parseNumber(str)
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, nil
}

func parseNumber(str string) (float64, error) {
	str = strings.TrimSpace(strings.ToLower(str))
	if len(str) == 0 {
		return 0, nil
	}
	lb := str[len(str)-1]
	if lb < '0' || lb > '9' {
		str = str[:len(str)-1]
	}
	n, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("Invalid rate value %v", str)
	}

	switch lb {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		n = n
	case 'k':
		n *= 1000
	case 'm':
		n *= 1000 * 1000
	case 'g':
		n *= 1000 * 1000 * 1000
	default:
		return 0, fmt.Errorf("Unsupported unit '%c' for rate", lb)
	}
	return n, nil
}
