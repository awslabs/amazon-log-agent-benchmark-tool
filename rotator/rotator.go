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
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Rotator interface {
	Rotate() (io.Writer, error)
}

type FileRotator struct {
	path string
	keep int

	current *os.File
}

func NewFileRotator(path string, keep int) *FileRotator {
	return &FileRotator{path: path, keep: keep}
}

func (fr *FileRotator) Rotate() (io.Writer, error) {
	log.Printf("Rotating %v", fr.path)
	if fr.current != nil {
		if err := fr.current.Close(); err != nil {
			return nil, fmt.Errorf("failed to close current file %v: %w", fr.path, err)
		}

		if err := fr.rotateFiles(); err != nil {
			return nil, fmt.Errorf("failed to rotate old log files: %w", err)
		}
	}

	f, err := os.Create(fr.path)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %v: %w", fr.path, err)
	}
	fr.current = f
	return f, nil
}

func (fr FileRotator) rotateFiles() error {
	for i := fr.keep - 1; i >= 0; i-- {
		f := fr.nthBackupPath(i)
		_, err := os.Stat(f)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		t := fr.nthBackupPath(i + 1)
		if err := os.Rename(f, t); err != nil {
			return fmt.Errorf("failed to move %v to %v: %w", f, t, err)
		}
	}
	return nil
}

func (fr FileRotator) nthBackupPath(n int) string {
	if n == 0 {
		return fr.path
	}
	ext := filepath.Ext(fr.path)
	name := strings.TrimSuffix(fr.path, ext)

	return fmt.Sprintf("%v.%v%v", name, n, ext)
}
