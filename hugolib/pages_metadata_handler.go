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

package hugolib

import (
	"github.com/gohugoio/hugo/common/herrors"
	"github.com/gohugoio/hugo/parser/metadecoders"
	"github.com/gohugoio/hugo/resources/page"
	"github.com/gohugoio/hugo/resources/resource"
)

const cascadeKey = "cascade"

type pagesMetadataHandler struct {
	dates *resource.Dates
}

func (m *pagesMetadataHandler) handleSection(
	parentCascade map[string]interface{},
	section *pageState) error {

	var err error

	if parentCascade, err = m.parseAndAssignMeta(true, parentCascade, section); err != nil {
		return err
	}

	var sectionDates *resource.Dates
	if section.IsHome() && resource.IsZeroDates(section) {
		// Calculate the home dates from the entire tree.
		m.dates = &resource.Dates{}
	} else if resource.IsZeroDates(section) {
		// Calculate the section dates from the section pages.
		sectionDates = &resource.Dates{}
	}

	for _, p := range section.pages {
		if parentCascade, err = m.parseAndAssignMeta(false, parentCascade, p.(*pageState)); err != nil {
			return err
		}

		if sectionDates != nil {
			sectionDates.UpdateDateAndLastmodIfAfter(p)
		}

		if m.dates != nil {
			m.dates.UpdateDateAndLastmodIfAfter(p)
		}

		// TODO(bep) cascade check sort
		for _, rp := range p.Resources().ByType(pageResourceType) {
			if _, err := m.parseAndAssignMeta(false, parentCascade, rp.(*pageState)); err != nil {
			}
		}
	}

	if sectionDates != nil {
		section.m.Dates = *sectionDates
	}

	// Now all metadata is set for the section and we can sort the pages.
	page.SortByDefault(section.pages)

	for _, p := range section.subSections {
		ps := p.(*pageState)

		if err := m.handleSection(parentCascade, ps); err != nil {
			return err
		}

	}

	// Now all metadata is set for the sections and we can sort them.
	page.SortByDefault(section.subSections)

	return nil

}

func (m *pagesMetadataHandler) parseAndAssignMeta(isSection bool, parentCascade map[string]interface{}, p *pageState) (map[string]interface{}, error) {
	var initErr error
	p.m.metaInit.Do(func() {
		var meta map[string]interface{}

		p.m.cascade = parentCascade

		if p.source.frontMatter.Val != nil {
			var err error
			meta, err = m.parseMeta(p.source)
			if err != nil {
				initErr = err
				return
			}

			if isSection {
				if c, found := meta[cascadeKey]; found {
					// Use this section's cascade for all children.
					p.m.cascade = c.(map[string]interface{})
					delete(meta, cascadeKey)
				}
			}
		} else {
			meta = make(map[string]interface{})
		}

		if p.m.cascade != nil {
			for k, v := range p.m.cascade {
				// TODO(bep) cascade case
				if k == cascadeKey {
					continue
				}
				if _, found := meta[k]; !found {
					meta[k] = v
				}
			}
		}

		if err := p.m.setMetadata(p, meta); err != nil {
			initErr = err
			return
		}

	})

	return p.m.cascade, initErr
}

func (m *pagesMetadataHandler) parseMeta(pc rawPageContent) (map[string]interface{}, error) {
	it := pc.frontMatter
	f := metadecoders.FormatFromFrontMatterType(it.Type)
	meta, err := metadecoders.Default.UnmarshalToMap(it.Val, f)
	if err != nil {
		if fe, ok := err.(herrors.FileError); ok {
			// TODO(bep) cascade check this in the browser
			return nil, herrors.ToFileErrorWithOffset(fe, pc.frontMatterLineNumber)
		} else {
			return nil, err
		}
	}

	return meta, nil
}
