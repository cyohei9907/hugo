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
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gohugoio/hugo/helpers"

	"github.com/gohugoio/hugo/resources/page"
)

// Used in the page cache to mark more than one hit for a given key.
var ambiguityFlag = &pageState{}

// PageCollections contains the page collections for a site.
type PageCollections struct {
	pageMap *pageMap

	// Lazy initialized page collections
	pages           *lazyPagesFactory
	regularPages    *lazyPagesFactory
	allPages        *lazyPagesFactory
	allRegularPages *lazyPagesFactory
}

// Pages returns all pages.
// This is for the current language only.
func (c *PageCollections) Pages() page.Pages {
	return c.pages.get()
}

// RegularPages returns all the regular pages.
// This is for the current language only.
func (c *PageCollections) RegularPages() page.Pages {
	return c.regularPages.get()
}

// AllPages returns all pages for all languages.
func (c *PageCollections) AllPages() page.Pages {
	return c.allPages.get()
}

// AllPages returns all regular pages for all languages.
func (c *PageCollections) AllRegularPages() page.Pages {
	return c.allRegularPages.get()
}

type lazyPagesFactory struct {
	pages page.Pages

	init    sync.Once
	factory page.PagesFactory
}

func (l *lazyPagesFactory) get() page.Pages {
	l.init.Do(func() {
		l.pages = l.factory()
	})
	return l.pages
}

func newLazyPagesFactory(factory page.PagesFactory) *lazyPagesFactory {
	return &lazyPagesFactory{factory: factory}
}

func newPageCollections(m *pageMap) *PageCollections {
	if m == nil {
		panic("must provide a pageMap")
	}

	c := &PageCollections{pageMap: m}

	c.pages = newLazyPagesFactory(func() page.Pages {
		return m.createListAllPages()
	})

	c.regularPages = newLazyPagesFactory(func() page.Pages {
		return c.findPagesByKindIn(page.KindPage, c.pages.get())
	})

	return c
}

// This is an adapter func for the old API with Kind as first argument.
// This is invoked when you do .Site.GetPage. We drop the Kind and fails
// if there are more than 2 arguments, which would be ambigous.
func (c *PageCollections) getPageOldVersion(ref ...string) (page.Page, error) {
	var refs []string
	for _, r := range ref {
		// A common construct in the wild is
		// .Site.GetPage "home" "" or
		// .Site.GetPage "home" "/"
		if r != "" && r != "/" {
			refs = append(refs, r)
		}
	}

	var key string

	if len(refs) > 2 {
		// This was allowed in Hugo <= 0.44, but we cannot support this with the
		// new API. This should be the most unusual case.
		return nil, fmt.Errorf(`too many arguments to .Site.GetPage: %v. Use lookups on the form {{ .Site.GetPage "/posts/mypage-md" }}`, ref)
	}

	if len(refs) == 0 || refs[0] == page.KindHome {
		key = "/"
	} else if len(refs) == 1 {
		if len(ref) == 2 && refs[0] == page.KindSection {
			// This is an old style reference to the "Home Page section".
			// Typically fetched via {{ .Site.GetPage "section" .Section }}
			// See https://github.com/gohugoio/hugo/issues/4989
			key = "/"
		} else {
			key = refs[0]
		}
	} else {
		key = refs[1]
	}

	key = filepath.ToSlash(key)
	if !strings.HasPrefix(key, "/") {
		key = "/" + key
	}

	return c.getPageNew(nil, key)
}

// TODO1 https://github.com/gohugoio/hugo/issues/6701 /Readme.md

// 	Only used in tests.
func (c *PageCollections) getPage(typ string, sections ...string) page.Page {
	refs := append([]string{typ}, path.Join(sections...))
	p, _ := c.getPageOldVersion(refs...)
	return p
}

func (c *PageCollections) getPageNew(context page.Page, ref string) (page.Page, error) {
	n, err := c.getContentNode(context, ref)
	if err != nil || n == nil || n.p == nil {
		return nil, err
	}
	return n.p, nil
}

func (c *PageCollections) getSectionOrPage(ref string) (*contentNode, string) {
	s, v, found := c.pageMap.sections.LongestPrefix(ref)

	m := c.pageMap
	filename := strings.TrimPrefix(strings.TrimPrefix(ref, s), "/")
	langSuffix := "." + m.s.Lang()

	// Trim both extension and any language code.
	name := helpers.PathNoExt(filename)
	name = strings.TrimSuffix(name, langSuffix)

	// These are reserved bundle names and will always be stored by their owning
	// folder name.
	name = strings.TrimSuffix(name, "/index")
	name = strings.TrimSuffix(name, "/_index")

	if !found {
		return nil, name
	}

	n := v.(*contentNode)

	if s == ref {
		// A section
		return n, name
	}

	// Check if it's a section with filename provided.
	if !n.p.File().IsZero() && n.p.File().LogicalName() == filename {
		return n, name
	}

	return m.getPage(s, name), name

}

func (c *PageCollections) getContentNode(context page.Page, ref string) (*contentNode, error) {
	inRef := ref
	if !strings.HasPrefix(ref, "/") && context != nil {
		// Try the page-relative path.
		var base string
		// TODO1 vs mount / Path
		if context.File().IsZero() {
			base = context.SectionsPath()
		} else {
			base = filepath.ToSlash(context.File().Dir())
		}
		ref = path.Join("/", strings.ToLower(base), ref)
	}

	ref = strings.ToLower(ref)
	if !strings.HasPrefix(ref, "/") {
		ref = "/" + ref
	}

	m := c.pageMap

	// It's either a section, a page in a section or a taxonomy node.
	// Start with the most likely:
	n, name := c.getSectionOrPage(ref)
	if n != nil {
		return n, nil
	}

	if !strings.HasPrefix(inRef, "/") {
		// Many people will have "post/foo.md" in their content files.
		if n, _ := c.getSectionOrPage("/" + inRef); n != nil {
			return n, nil
		}
	}

	// Check if it's a taxonomy node
	s, v, found := m.taxonomies.LongestPrefix(ref)
	if found {
		if !m.onSameLevel(ref, s) {
			return nil, nil
		}
		return v.(*contentNode), nil
	}

	// ref/relref supports this potentially ambigous format
	// TODO(bep) consider storing the value.
	s = m.pagesByName.Get(name)
	if s != "" {
		if s == ambiguityKey {
			return nil, fmt.Errorf("page reference %q is ambiguous", ref)
		}
		v, _ := m.pages.Get(s)
		return v.(*contentNode), nil
	}

	return nil, nil
}

func (*PageCollections) findPagesByKindIn(kind string, inPages page.Pages) page.Pages {
	var pages page.Pages
	for _, p := range inPages {
		if p.Kind() == kind {
			pages = append(pages, p)
		}
	}
	return pages
}

func (c *PageCollections) clearResourceCacheForPage(page *pageState) {
	if len(page.resources) > 0 {
		page.s.ResourceSpec.DeleteCacheByPrefix(page.targetPaths().SubResourceBaseTarget)
	}
}
