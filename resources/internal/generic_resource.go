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

package internal

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/gohugoio/hugo/common/hugio"
	"github.com/gohugoio/hugo/common/loggers"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/media"
	"github.com/gohugoio/hugo/resources/page"
	"github.com/gohugoio/hugo/resources/resource"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type Spec interface {
	Permalink(link string) string
	URLizeFilename(name string) string
	GetBasePath(isRelative bool) string
	PublishFs() afero.Fs
	Logger() *loggers.Logger
}

var NoData = make(map[string]interface{})

func NewGenericResource(
	spec Spec,
	sourceFs afero.Fs,
	lazyPublish bool,
	openReadSeekerCloser resource.OpenReadSeekCloser,
	targetPathBaseDirs []string,
	targetPathBuilder func() page.TargetPaths,
	osFileInfo os.FileInfo,
	sourceFilename,
	baseFilename string,
	mediaType media.Type) *GenericResource {

	if osFileInfo != nil && osFileInfo.IsDir() {
		panic(fmt.Sprintf("dirs not supported resource types: %v", osFileInfo))
	}

	// This value is used both to construct URLs and file paths, but start
	// with a Unix-styled path.
	baseFilename = helpers.ToSlashTrimLeading(baseFilename)
	fpath, fname := path.Split(baseFilename)

	var resourceType string
	if mediaType.MainType == "image" {
		resourceType = mediaType.MainType
	} else {
		resourceType = mediaType.SubType
	}

	pathDescriptor := ResourcePathDescriptor{
		BaseTargetPathDirs: helpers.UniqueStringsReuse(targetPathBaseDirs),
		TargetPathBuilder:  targetPathBuilder,
		RelTargetDirFile:   DirFile{Dir: fpath, File: fname},
	}

	var po *PublishOnce
	if lazyPublish {
		po = &PublishOnce{Logger: spec.Logger()}
	}

	return &GenericResource{
		openReadSeekerCloser:   openReadSeekerCloser,
		PublishOnce:            po,
		ResourcePathDescriptor: pathDescriptor,
		sourceFs:               sourceFs,
		osFileInfo:             osFileInfo,
		sourceFilename:         sourceFilename,
		mediaType:              mediaType,
		resourceType:           resourceType,
		spec:                   spec,
		params:                 make(map[string]interface{}),
		name:                   baseFilename,
		title:                  baseFilename,
		resourceContent:        &resourceContent{},
		resourceHash:           &resourceHash{},
	}
}

// GenericResource represents a generic linkable resource.
type GenericResource struct {
	CommonResource
	ResourcePathDescriptor // TODO(bep) image check the visibility of these fields

	title  string
	name   string
	params map[string]interface{}

	// Absolute filename to the source, including any content folder path.
	// Note that this is absolute in relation to the filesystem it is stored in.
	// It can be a base path filesystem, and then this filename will not match
	// the path to the file on the real filesystem.
	sourceFilename string

	// Will be set if this resource is backed by something other than a file.
	openReadSeekerCloser resource.OpenReadSeekCloser

	// A hash of the source content. Is only calculated in caching situations.
	*resourceHash

	// This may be set to tell us to look in another filesystem for this resource.
	// We, by default, use the sourceFs filesystem in the spec below.
	sourceFs afero.Fs

	spec Spec

	resourceType string
	mediaType    media.Type

	osFileInfo os.FileInfo

	// We create copies of this struct, so this needs to be a pointer.
	*resourceContent

	// May be set to signal lazy/delayed publishing.
	*PublishOnce
}

func (l *GenericResource) Data() interface{} {
	return NoData
}

func (l *GenericResource) Content() (interface{}, error) {
	if err := l.initContent(); err != nil {
		return nil, err
	}

	return l.content, nil
}

func (l *GenericResource) ReadSeekCloser() (hugio.ReadSeekCloser, error) {
	if l.openReadSeekerCloser != nil {
		return l.openReadSeekerCloser()
	}

	f, err := l.getSourceFs().Open(l.sourceFilename)
	if err != nil {
		return nil, err
	}
	return f, nil

}

func (l *GenericResource) MediaType() media.Type {
	return l.mediaType
}

// Implement the Cloner interface.
func (l GenericResource) WithNewBase(base string) resource.Resource {
	l.BaseOffset = base
	l.resourceContent = &resourceContent{}
	return &l
}

// TODO(bep) image check visibility
type CommonResource struct {
}

// Slice is not meant to be used externally. It's a bridge function
// for the template functions. See collections.Slice.
func (CommonResource) Slice(in interface{}) (interface{}, error) {
	switch items := in.(type) {
	case resource.Resources:
		return items, nil
	case []interface{}:
		groups := make(resource.Resources, len(items))
		for i, v := range items {
			g, ok := v.(resource.Resource)
			if !ok {
				return nil, fmt.Errorf("type %T is not a Resource", v)
			}
			groups[i] = g
		}
		return groups, nil
	default:
		return nil, fmt.Errorf("invalid slice type %T", items)
	}
}

func (l *GenericResource) initHash() error {
	var err error
	l.hashInit.Do(func() {
		var hash string
		var f hugio.ReadSeekCloser
		f, err = l.ReadSeekCloser()
		if err != nil {
			err = errors.Wrap(err, "failed to open source file")
			return
		}
		defer f.Close()

		hash, err = helpers.MD5FromFileFast(f)
		if err != nil {
			return
		}
		l.hash = hash

	})

	return err
}

func (l *GenericResource) initContent() error {
	var err error
	l.contentInit.Do(func() {
		var r hugio.ReadSeekCloser
		r, err = l.ReadSeekCloser()
		if err != nil {
			return
		}
		defer r.Close()

		var b []byte
		b, err = ioutil.ReadAll(r)
		if err != nil {
			return
		}

		l.content = string(b)

	})

	return err
}

func (l *GenericResource) getSourceFs() afero.Fs {
	return l.sourceFs
}

func (l *GenericResource) publishIfNeeded() {
	if l.PublishOnce != nil {
		l.PublishOnce.publish(l)
	}
}

func (l *GenericResource) Permalink() string {
	l.publishIfNeeded()
	return l.spec.Permalink(l.relPermalinkForRel(l.RelTargetDirFile.path(), true))
}

func (l *GenericResource) RelPermalink() string {
	l.publishIfNeeded()
	return l.relPermalinkFor(l.RelTargetDirFile.path())
}

func (l *GenericResource) Key() string {
	return l.RelTargetDirFile.path()
}

func (l *GenericResource) relPermalinkFor(target string) string {
	return l.relPermalinkForRel(target, false)

}
func (l *GenericResource) permalinkFor(target string) string {
	return l.spec.Permalink(l.relPermalinkForRel(target, true))

}
func (l *GenericResource) relTargetPathsFor(target string) []string {
	return l.relTargetPathsForRel(target)
}

func (l *GenericResource) relTargetPaths() []string {
	return l.relTargetPathsForRel(l.TargetPath())
}

func (l *GenericResource) Name() string {
	return l.name
}

func (l *GenericResource) Title() string {
	return l.title
}

func (l *GenericResource) Params() map[string]interface{} {
	return l.params
}

func (l *GenericResource) setTitle(title string) {
	l.title = title
}

func (l *GenericResource) setName(name string) {
	l.name = name
}

func (l *GenericResource) updateParams(params map[string]interface{}) {
	if l.params == nil {
		l.params = params
		return
	}

	// Sets the params not already set
	for k, v := range params {
		if _, found := l.params[k]; !found {
			l.params[k] = v
		}
	}
}

func (l *GenericResource) relPermalinkForRel(rel string, isAbs bool) string {
	return l.spec.URLizeFilename(l.relTargetPathForRel(rel, false, isAbs, true))
}

func (l *GenericResource) relTargetPathsForRel(rel string) []string {
	if len(l.BaseTargetPathDirs) == 0 {
		return []string{l.relTargetPathForRelAndBasePath(rel, "", false, false)}
	}

	var targetPaths = make([]string, len(l.BaseTargetPathDirs))
	for i, dir := range l.BaseTargetPathDirs {
		targetPaths[i] = l.relTargetPathForRelAndBasePath(rel, dir, false, false)
	}
	return targetPaths
}

func (l *GenericResource) relTargetPathForRel(rel string, addBaseTargetPath, isAbs, isURL bool) string {
	if addBaseTargetPath && len(l.BaseTargetPathDirs) > 1 {
		panic("multiple baseTargetPathDirs")
	}
	var basePath string
	if addBaseTargetPath && len(l.BaseTargetPathDirs) > 0 {
		basePath = l.BaseTargetPathDirs[0]
	}

	return l.relTargetPathForRelAndBasePath(rel, basePath, isAbs, isURL)
}

func (l *GenericResource) createBasePath(rel string, isURL bool) string {
	if l.TargetPathBuilder == nil {
		return rel
	}
	tp := l.TargetPathBuilder()

	if isURL {
		return path.Join(tp.SubResourceBaseLink, rel)
	}

	// TODO(bep) path
	return path.Join(filepath.ToSlash(tp.SubResourceBaseTarget), rel)
}

func (l *GenericResource) relTargetPathForRelAndBasePath(rel, basePath string, isAbs, isURL bool) string {
	rel = l.createBasePath(rel, isURL)

	if basePath != "" {
		rel = path.Join(basePath, rel)
	}

	if l.BaseOffset != "" {
		rel = path.Join(l.BaseOffset, rel)
	}

	if isURL {
		bp := l.spec.GetBasePath(!isAbs)
		if bp != "" {
			rel = path.Join(bp, rel)
		}
	}

	if len(rel) == 0 || rel[0] != '/' {
		rel = "/" + rel
	}

	return rel
}

func (l *GenericResource) ResourceType() string {
	return l.resourceType
}

func (l *GenericResource) String() string {
	return fmt.Sprintf("Resource(%s: %s)", l.resourceType, l.name)
}

func (l *GenericResource) Publish() error {
	fr, err := l.ReadSeekCloser()
	if err != nil {
		return err
	}
	defer fr.Close()

	fw, err := helpers.OpenFilesForWriting(l.spec.PublishFs(), l.targetFilenames()...)
	if err != nil {
		return err
	}
	defer fw.Close()

	_, err = io.Copy(fw, fr)
	return err
}

// Path is stored with Unix style slashes.
func (l *GenericResource) TargetPath() string {
	return l.RelTargetDirFile.path()
}

func (l *GenericResource) targetFilenames() []string {
	paths := l.relTargetPaths()
	for i, p := range paths {
		paths[i] = filepath.Clean(p)
	}
	return paths
}

type ResourcePathDescriptor struct {
	// The relative target directory and filename.
	RelTargetDirFile DirFile

	// Callback used to construct a target path relative to its owner.
	TargetPathBuilder func() page.TargetPaths

	// This will normally be the same as above, but this will only apply to publishing
	// of resources. It may be mulltiple values when in multihost mode.
	BaseTargetPathDirs []string

	// BaseOffset is set when the output format's path has a offset, e.g. for AMP.
	BaseOffset string
}

type resourceContent struct {
	content     string
	contentInit sync.Once
}

type resourceHash struct {
	hash     string
	hashInit sync.Once
}

type DirFile struct {
	// This is the directory component with Unix-style slashes.
	Dir string
	// This is the file component.
	File string
}

func (d DirFile) path() string {
	return path.Join(d.Dir, d.File)
}

type PublishOnce struct {
	publisherInit sync.Once
	publisherErr  error
	Logger        *loggers.Logger
}

func (l *PublishOnce) publish(s resource.Source) error {
	l.publisherInit.Do(func() {
		l.publisherErr = s.Publish()
		if l.publisherErr != nil {
			l.Logger.ERROR.Printf("failed to publish Resource: %s", l.publisherErr)
		}
	})
	return l.publisherErr
}
