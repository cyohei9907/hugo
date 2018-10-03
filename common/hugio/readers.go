// Copyright 2018 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hugio

import (
	"bytes"
	"io"
	"strings"
)

// ReadSeeker wraps io.Reader and io.Seeker.
type ReadSeeker interface {
	io.Reader
	io.Seeker
}

// ReadSeekCloser is implemented by afero.File. We use this as the common type for
// content in Resource objects, even for strings.
type ReadSeekCloser interface {
	ReadSeeker
	io.Closer
}

// ReadSeekerNoOpCloser implements ReadSeekCloser by doing nothing in Close.
type ReadSeekerNoOpCloser struct {
	ReadSeeker
}

// Close does nothing.
func (r ReadSeekerNoOpCloser) Close() error {
	return nil
}

// NewReadSeekerNoOpCloser creates a new ReadSeekerNoOpCloser with the given ReadSeeker.
func NewReadSeekerNoOpCloser(r ReadSeeker) ReadSeekerNoOpCloser {
	return ReadSeekerNoOpCloser{r}
}

// NewReadSeekerNoOpCloserFromString uses strings.NewReader to create a new ReadSeekerNoOpCloser
// from the given string.
func NewReadSeekerNoOpCloserFromString(content string) ReadSeekerNoOpCloser {
	return ReadSeekerNoOpCloser{strings.NewReader(content)}
}

// LineCountingReader wraps io.Reader and counts lines, i.e. increments when
// it sees a '\n'.
type LineCountingReader struct {
	r     io.Reader
	Count int
}

// NewLineCountingReader creates a new reader that counts the lines read from r.
func NewLineCountingReader(r io.Reader) *LineCountingReader {
	return &LineCountingReader{r: r}
}

func (l *LineCountingReader) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	if n > 0 {
		l.Count += bytes.Count(p[:n], []byte{'\n'})
	}
	return
}
