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
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

type Opt func(g *Generator)

func OptTimeLayout(tf string) func(g *Generator) {
	return func(g *Generator) {
		g.timeFormat = tf
	}
}

func OptRate(rate float64) func(g *Generator) {
	return func(g *Generator) {
		g.rate = rate
	}
}

func OptLines(lines []string) func(g *Generator) {
	return func(g *Generator) {
		g.buf = lines
	}
}

type Generators []*Generator

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

func NewFixed(line string, dests []io.Writer, opts ...Opt) Generators {
	var gens Generators
	opts = append(opts, OptLines([]string{line}))
	for _, dest := range dests {
		gens = append(gens, newGenerator(dest, opts...))
	}
	return gens
}

func NewGeneratorFromFile(path string, dest io.Writer, opts ...Opt) (*Generator, error) {
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
	opts = append(opts, OptLines(lines))

	return newGenerator(dest, opts...), nil
}

type Generator struct {
	dest           io.Writer
	rate           float64
	buf            []string
	idx            int
	done           chan struct{}
	rateCh         chan float64
	timeFormat     string
	rotateSize     int64
	rotateDuratoin time.Duration

	rand *rand.Rand
}

func newGenerator(dest io.Writer, opts ...Opt) *Generator {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	g := &Generator{
		dest:       dest,
		done:       make(chan struct{}),
		rateCh:     make(chan float64),
		rand:       r,
		timeFormat: time.StampNano,
	}
	for _, opt := range opts {
		opt(g)
	}
	g.Start()
	return g
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
					tn = tn.Add(g.delay())
					_, err := fmt.Fprintf(g.dest, "%v %s\n", now.Format(g.timeFormat), g.nextLine())
					if err != nil {
						log.Printf("Failed to write to %v with error: %v", g.dest, err)
					}
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

func (g *Generator) delay() time.Duration {
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
