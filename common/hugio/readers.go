// Copyright 2019 The Hugo Authors. All rights reserved.
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
// TODO(bep) rename this and simila to ReadSeekerNopCloser, naming used in stdlib, which kind of makes sense.
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

type multiReadCloser struct {
	io.Reader
	closers []io.ReadCloser
}

func (m multiReadCloser) Close() error {
	var err error
	for _, c := range m.closers {
		if closeErr := c.Close(); err != nil {
			err = closeErr
		}
	}
	return err
}

// NewMultiReadCloser returns a io.ReadCloser that's the logical concatenation of
// the provided input readers.
func NewMultiReadCloser(readClosers ...io.ReadCloser) io.ReadCloser {
	readers := make([]io.Reader, len(readClosers))
	for i, r := range readClosers {
		readers[i] = r
	}
	return multiReadCloser{Reader: io.MultiReader(readers...), closers: readClosers}
}
