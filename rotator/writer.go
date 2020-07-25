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
package rotator

import (
	"io"
	"time"
)

type Config struct {
	Keep     int
	Duration time.Duration
	Size     int64
}

type Writer struct {
	w io.Writer
	r Rotator
	c Config

	rt   time.Time
	size int64
}

func NewWriter(r Rotator, c Config) (*Writer, error) {
	w := &Writer{r: r, c: c}
	if err := w.Rotate(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) Write(b []byte) (int, error) {
	now := time.Now()

	if w.c.Duration > 0 {
		if w.rt.IsZero() {
			w.rt = now.Add(w.c.Duration)
		}

		if now.After(w.rt) {
			if err := w.Rotate(); err != nil {
				return 0, err
			}
		}
	}

	if w.c.Size > 0 && w.size+int64(len(b)) > w.c.Size {
		if err := w.Rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.w.Write(b)
	w.size += int64(n)
	return n, err
}

func (w *Writer) Rotate() error {
	nw, err := w.r.Rotate()
	if err != nil {
		return err
	}
	w.w = nw
	w.size = 0
	w.rt = time.Time{}
	return nil
}
