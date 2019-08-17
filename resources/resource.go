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

package resources

import (
	"os"
	"path"
	"sync"

	"github.com/gohugoio/hugo/common/loggers"
	"github.com/gohugoio/hugo/resources/page"
	"github.com/gohugoio/hugo/resources/resource"

	"github.com/spf13/afero"

	"github.com/gohugoio/hugo/source"
)

var (
	// TODO(bep) image
	/*
		_ resource.ContentResource        = (*GenericResource)(nil)
		_ resource.ReadSeekCloserResource = (*GenericResource)(nil)
		_ resource.Resource               = (*GenericResource)(nil)
		_ resource.Source                 = (*GenericResource)(nil)
		_ resource.Cloner                 = (*GenericResource)(nil)
		_ permalinker                     = (*GenericResource)(nil)
		_ collections.Slicer              = (*GenericResource)(nil)
		_ resource.Identifier             = (*GenericResource)(nil)
	*/

	_ resource.ResourcesLanguageMerger = (*resource.Resources)(nil)
)

var noData = make(map[string]interface{})

type permalinker interface {
	relPermalinkFor(target string) string
	permalinkFor(target string) string
	relTargetPathsFor(target string) []string
	relTargetPaths() []string
	TargetPath() string
}

type Spec struct {
}

type ResourceSourceDescriptor struct {
	// TargetPaths is a callback to fetch paths's relative to its owner.
	TargetPaths func() page.TargetPaths

	// Need one of these to load the resource content.
	SourceFile         source.File
	OpenReadSeekCloser resource.OpenReadSeekCloser

	FileInfo os.FileInfo

	// If OpenReadSeekerCloser is not set, we use this to open the file.
	SourceFilename string

	Fs afero.Fs

	// The relative target filename without any language code.
	RelTargetFilename string

	// Any base paths prepended to the target path. This will also typically be the
	// language code, but setting it here means that it should not have any effect on
	// the permalink.
	// This may be several values. In multihost mode we may publish the same resources to
	// multiple targets.
	TargetBasePaths []string

	// Delay publishing until either Permalink or RelPermalink is called. Maybe never.
	LazyPublish bool
}

func (r ResourceSourceDescriptor) Filename() string {
	if r.SourceFile != nil {
		return r.SourceFile.Filename()
	}
	return r.SourceFilename
}

type dirFile struct {
	// This is the directory component with Unix-style slashes.
	dir string
	// This is the file component.
	file string
}

func (d dirFile) path() string {
	return path.Join(d.dir, d.file)
}

type resourcePathDescriptor struct {
	// The relative target directory and filename.
	relTargetDirFile dirFile

	// Callback used to construct a target path relative to its owner.
	targetPathBuilder func() page.TargetPaths

	// This will normally be the same as above, but this will only apply to publishing
	// of resources. It may be mulltiple values when in multihost mode.
	baseTargetPathDirs []string

	// baseOffset is set when the output format's path has a offset, e.g. for AMP.
	baseOffset string
}

type resourceContent struct {
	content     string
	contentInit sync.Once
}

type resourceHash struct {
	hash     string
	hashInit sync.Once
}

type publishOnce struct {
	publisherInit sync.Once
	publisherErr  error
	logger        *loggers.Logger
}

func (l *publishOnce) publish(s resource.Source) error {
	l.publisherInit.Do(func() {
		l.publisherErr = s.Publish()
		if l.publisherErr != nil {
			l.logger.ERROR.Printf("failed to publish Resource: %s", l.publisherErr)
		}
	})
	return l.publisherErr
}
