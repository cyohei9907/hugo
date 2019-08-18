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
	"errors"
	"fmt"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gohugoio/hugo/helpers"

	"github.com/gohugoio/hugo/cache/filecache"
	"github.com/gohugoio/hugo/common/loggers"
	"github.com/gohugoio/hugo/media"
	"github.com/gohugoio/hugo/output"
	"github.com/gohugoio/hugo/resources/images"
	"github.com/gohugoio/hugo/resources/page"
	"github.com/gohugoio/hugo/resources/resource"
	"github.com/gohugoio/hugo/tpl"
	"github.com/spf13/afero"
)

func NewSpec(
	s *helpers.PathSpec,
	fileCaches filecache.Caches,
	logger *loggers.Logger,
	outputFormats output.Formats,
	mimeTypes media.Types) (*Spec, error) {

	imgConfig, err := images.DecodeConfig(s.Cfg.GetStringMap("imaging"))
	if err != nil {
		return nil, err
	}

	imaging := &images.ImageProcessor{Cfg: imgConfig}

	if logger == nil {
		logger = loggers.NewErrorLogger()
	}

	permalinks, err := page.NewPermalinkExpander(s)
	if err != nil {
		return nil, err
	}

	rs := &Spec{PathSpec: s,
		Logger:        logger,
		imaging:       imaging,
		MediaTypes:    mimeTypes,
		OutputFormats: outputFormats,
		Permalinks:    permalinks,
		FileCaches:    fileCaches,
		imageCache: newImageCache(
			fileCaches.ImageCache(),

			s,
		)}

	rs.ResourceCache = newResourceCache(rs)

	return rs, nil

}

type Spec struct {
	*helpers.PathSpec

	MediaTypes    media.Types
	OutputFormats output.Formats

	Logger *loggers.Logger

	TextTemplates tpl.TemplateParseFinder

	Permalinks page.PermalinkExpander

	// Holds default filter settings etc.
	imaging *images.ImageProcessor

	imageCache    *imageCache
	ResourceCache *ResourceCache
	FileCaches    filecache.Caches
}

func (r *Spec) New(fd ResourceSourceDescriptor) (resource.Resource, error) {
	return r.newResourceFor(fd)
}

func (r *Spec) CacheStats() string {
	r.imageCache.mu.RLock()
	defer r.imageCache.mu.RUnlock()

	s := fmt.Sprintf("Cache entries: %d", len(r.imageCache.store))

	count := 0
	for k := range r.imageCache.store {
		if count > 5 {
			break
		}
		s += "\n" + k
		count++
	}

	return s
}

func (r *Spec) ClearCaches() {
	r.imageCache.clear()
	r.ResourceCache.clear()
}

func (r *Spec) DeleteCacheByPrefix(prefix string) {
	r.imageCache.deleteByPrefix(prefix)
}

// TODO(bep) unify
func (r *Spec) IsInImageCache(key string) bool {
	// This is used for cache pruning. We currently only have images, but we could
	// imagine expanding on this.
	return r.imageCache.isInCache(key)
}

func (s *Spec) String() string {
	return "spec"
}

// TODO(bep) clean up below
func (r *Spec) newGenericResource(sourceFs afero.Fs,
	targetPathBuilder func() page.TargetPaths,
	osFileInfo os.FileInfo,
	sourceFilename,
	baseFilename string,
	mediaType media.Type) *genericResource {
	return r.newGenericResourceWithBase(
		sourceFs,
		false,
		nil,
		nil,
		targetPathBuilder,
		osFileInfo,
		sourceFilename,
		baseFilename,
		mediaType,
	)

}

func (r *Spec) newGenericResourceWithBase(
	sourceFs afero.Fs,
	lazyPublish bool,
	openReadSeekerCloser resource.OpenReadSeekCloser,
	targetPathBaseDirs []string,
	targetPathBuilder func() page.TargetPaths,
	osFileInfo os.FileInfo,
	sourceFilename,
	baseFilename string,
	mediaType media.Type) *genericResource {

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

	pathDescriptor := &resourcePathDescriptor{
		baseTargetPathDirs: helpers.UniqueStringsReuse(targetPathBaseDirs),
		targetPathBuilder:  targetPathBuilder,
		relTargetDirFile:   dirFile{dir: fpath, file: fname},
	}

	var po *publishOnce
	if lazyPublish {
		po = &publishOnce{logger: r.Logger}
	}

	gfi := &resourceFileInfo{
		fi:                   osFileInfo,
		openReadSeekerCloser: openReadSeekerCloser,
		sourceFs:             sourceFs,
		sourceFilename:       sourceFilename,
		h:                    &resourceHash{},
	}

	return &genericResource{
		resourceFileInfo:       gfi,
		publishOnce:            po,
		resourcePathDescriptor: pathDescriptor,
		mediaType:              mediaType,
		resourceType:           resourceType,
		spec:                   r,
		params:                 make(map[string]interface{}),
		name:                   baseFilename,
		title:                  baseFilename,
		resourceContent:        &resourceContent{},
	}
}

func (r *Spec) newResource(sourceFs afero.Fs, fd ResourceSourceDescriptor) (resource.Resource, error) {
	fi := fd.FileInfo
	var sourceFilename string

	if fd.OpenReadSeekCloser != nil {
	} else if fd.SourceFilename != "" {
		var err error
		fi, err = sourceFs.Stat(fd.SourceFilename)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		sourceFilename = fd.SourceFilename
	} else {
		sourceFilename = fd.SourceFile.Filename()
	}

	if fd.RelTargetFilename == "" {
		fd.RelTargetFilename = sourceFilename
	}

	ext := strings.ToLower(filepath.Ext(fd.RelTargetFilename))
	mimeType, found := r.MediaTypes.GetFirstBySuffix(strings.TrimPrefix(ext, "."))
	// TODO(bep) we need to handle these ambigous types better, but in this context
	// we most likely want the application/xml type.
	if mimeType.Suffix() == "xml" && mimeType.SubType == "rss" {
		mimeType, found = r.MediaTypes.GetByType("application/xml")
	}

	if !found {
		// A fallback. Note that mime.TypeByExtension is slow by Hugo standards,
		// so we should configure media types to avoid this lookup for most
		// situations.
		mimeStr := mime.TypeByExtension(ext)
		if mimeStr != "" {
			mimeType, _ = media.FromStringAndExt(mimeStr, ext)
		}
	}

	gr := r.newGenericResourceWithBase(
		sourceFs,
		fd.LazyPublish,
		fd.OpenReadSeekCloser,
		fd.TargetBasePaths,
		fd.TargetPaths,
		fi,
		sourceFilename,
		fd.RelTargetFilename,
		mimeType)

	if mimeType.MainType == "image" {
		imgFormat, ok := images.ImageFormatFromExt(ext)
		if !ok {
			// This allows SVG etc. to be used as resources. They will not have the methods of the Image, but
			// that would not (currently) have worked.
			return gr, nil
		}

		return &imageResource{
			Image:        images.NewImage(imgFormat, r.imaging, nil, gr),
			baseResource: gr,
		}, nil
	}
	return gr, nil

}

func (r *Spec) newResourceFor(fd ResourceSourceDescriptor) (resource.Resource, error) {
	if fd.OpenReadSeekCloser == nil {
		if fd.SourceFile != nil && fd.SourceFilename != "" {
			return nil, errors.New("both SourceFile and AbsSourceFilename provided")
		} else if fd.SourceFile == nil && fd.SourceFilename == "" {
			return nil, errors.New("either SourceFile or AbsSourceFilename must be provided")
		}
	}

	if fd.RelTargetFilename == "" {
		fd.RelTargetFilename = fd.Filename()
	}

	if len(fd.TargetBasePaths) == 0 {
		// If not set, we publish the same resource to all hosts.
		fd.TargetBasePaths = r.MultihostTargetBasePaths
	}

	return r.newResource(fd.Fs, fd)
}
