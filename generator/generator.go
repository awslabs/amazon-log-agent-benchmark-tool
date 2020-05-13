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
package generator

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"time"
)

const timeFormat = time.StampNano

type Generator struct {
	dest   io.Writer
	rate   float64
	buf    []string
	idx    int
	done   chan struct{}
	rateCh chan float64

	rand *rand.Rand
}

type Generators []*Generator

func NewFixed(line string, logfiles []string) (Generators, error) {
	var gens Generators
	for _, path := range logfiles {
		out, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create log file at %v with error: %v", path, err)
		}
		gen := NewFixedGenerator(line, out, 0)
		gen.Start()
		gens = append(gens, gen)
	}
	return gens, nil
}

func (gs Generators) SetRate(r float64) {
	for _, gen := range gs {
		gen.SetRate(r)
	}
}

func (gs Generators) Stop() {
	for _, gen := range gs {
		gen.Stop()
	}
}

func NewGeneratorFromFile(path string, dest io.Writer, rate float64) (*Generator, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return &Generator{buf: lines, dest: dest, rate: rate, done: make(chan struct{}), rateCh: make(chan float64)}, nil
}

func NewFixedGenerator(line string, dest io.Writer, rate float64) *Generator {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Generator{
		buf:    []string{line},
		dest:   dest,
		rate:   rate,
		done:   make(chan struct{}),
		rateCh: make(chan float64),
		rand:   r,
	}
}

func (g *Generator) SetRate(r float64) {
	g.rateCh <- r
}

func (g *Generator) Start() {
	go func() {
		tn := time.Now()
		t := time.NewTimer(0)
		<-t.C
		for {
			select {
			case now := <-t.C:
				for {
					d := g.timeBetweenLine()
					fmt.Fprintf(g.dest, "%v %s\n", now.Format(timeFormat), g.nextLine())
					tn = tn.Add(d)
					if tn.After(now) {
						t.Reset(tn.Sub(now))
						break
					}
				}
			case r := <-g.rateCh:
				t.Stop()
				g.rate = r
				tn = time.Now()
				t.Reset(0)
			case <-g.done:
				t.Stop()
				return
			}
		}
	}()
}

func (g *Generator) Stop() {
	g.done <- struct{}{}
}

func (g *Generator) timeBetweenLine() time.Duration {
	if g.rate == 0 {
		return time.Duration(math.MaxInt64)
	}
	ts := g.rand.ExpFloat64() / (g.rate * 100) * float64(time.Second) * 100
	return time.Duration(ts)
}

func (g *Generator) nextLine() string {
	l := g.buf[g.idx]
	g.idx++
	g.idx %= len(g.buf)
	return l
}
