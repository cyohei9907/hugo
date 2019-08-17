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

package resource_common

import (
	"os"

	"github.com/gohugoio/hugo/cache/filecache"
	"github.com/gohugoio/hugo/common/loggers"
	"github.com/gohugoio/hugo/helpers"
	"github.com/gohugoio/hugo/media"
	"github.com/gohugoio/hugo/output"
	"github.com/gohugoio/hugo/resources/internal"
	"github.com/gohugoio/hugo/resources/page"
	"github.com/gohugoio/hugo/resources/resource"
	"github.com/gohugoio/hugo/tpl"
	"github.com/spf13/afero"
)

type Spec struct {
	*helpers.PathSpec

	MediaTypes    media.Types
	OutputFormats output.Formats

	logger *loggers.Logger

	TextTemplates tpl.TemplateParseFinder

	Permalinks page.PermalinkExpander

	// Holds default filter settings etc.
	imaging *Imaging

	//imageCache    *imageCache
	ResourceCache *ResourceCache
	FileCaches    filecache.Caches
}

func NewSpec(
	s *helpers.PathSpec,
	fileCaches filecache.Caches,
	logger *loggers.Logger,
	outputFormats output.Formats,
	mimeTypes media.Types) (*Spec, error) {

	// TODO(bep) image
	/*imaging, err := decodeImaging(s.Cfg.GetStringMap("imaging"))
	if err != nil {
		return nil, err
	}*/

	if logger == nil {
		logger = loggers.NewErrorLogger()
	}

	permalinks, err := page.NewPermalinkExpander(s)
	if err != nil {
		return nil, err
	}

	rs := &Spec{PathSpec: s,
		logger: logger,
		// imaging:       &imaging,
		MediaTypes:    mimeTypes,
		OutputFormats: outputFormats,
		Permalinks:    permalinks,
		FileCaches:    fileCaches,
		/*		imageCache: newImageCache(
				fileCaches.ImageCache(),

				s,*/
	}

	rs.ResourceCache = newResourceCache(rs)

	return rs, nil

}

func (r *Spec) Logger() *loggers.Logger {
	return r.logger
}

// TODO(bep) clean up below
func (r *Spec) newGenericResource(sourceFs afero.Fs,
	targetPathBuilder func() page.TargetPaths,
	osFileInfo os.FileInfo,
	sourceFilename,
	baseFilename string,
	mediaType media.Type) *internal.GenericResource {
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
	mediaType media.Type) *internal.GenericResource {

	return internal.NewGenericResource(
		r,
		sourceFs,
		lazyPublish,
		openReadSeekerCloser,
		targetPathBaseDirs,
		targetPathBuilder,
		osFileInfo,
		sourceFilename,
		baseFilename,
		mediaType,
	)

}
