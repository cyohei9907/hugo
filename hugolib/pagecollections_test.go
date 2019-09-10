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
	"math/rand"
	"path"
	"path/filepath"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"github.com/gohugoio/hugo/resources/page"

	"github.com/gohugoio/hugo/deps"
)

const pageCollectionsPageTemplate = `---
title: "%s"
categories:
- Hugo
---
# Doc
`

func BenchmarkGetPage(b *testing.B) {
	var (
		cfg, fs = newTestCfg()
		r       = rand.New(rand.NewSource(time.Now().UnixNano()))
	)

	for i := 0; i < 10; i++ {
		for j := 0; j < 100; j++ {
			writeSource(b, fs, filepath.Join("content", fmt.Sprintf("sect%d", i), fmt.Sprintf("page%d.md", j)), "CONTENT")
		}
	}

	s := buildSingleSite(b, deps.DepsCfg{Fs: fs, Cfg: cfg}, BuildCfg{SkipRender: true})

	pagePaths := make([]string, b.N)

	for i := 0; i < b.N; i++ {
		pagePaths[i] = fmt.Sprintf("sect%d", r.Intn(10))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		home, _ := s.getPageNew(nil, "/")
		if home == nil {
			b.Fatal("Home is nil")
		}

		p, _ := s.getPageNew(nil, pagePaths[i])
		if p == nil {
			b.Fatal("Section is nil")
		}

	}
}

func createGetPageRegularBenchmarkSite(t testing.TB) *Site {

	var (
		c       = qt.New(t)
		cfg, fs = newTestCfg()
	)

	pc := func(title string) string {
		return fmt.Sprintf(pageCollectionsPageTemplate, title)
	}

	for i := 0; i < 10; i++ {
		for j := 0; j < 100; j++ {
			content := pc(fmt.Sprintf("Title%d_%d", i, j))
			writeSource(c, fs, filepath.Join("content", fmt.Sprintf("sect%d", i), fmt.Sprintf("page%d.md", j)), content)
		}
	}

	return buildSingleSite(c, deps.DepsCfg{Fs: fs, Cfg: cfg}, BuildCfg{SkipRender: true})

}

func TestBenchmarkGetPageRegular(t *testing.T) {
	c := qt.New(t)
	s := createGetPageRegularBenchmarkSite(t)

	for i := 0; i < 10; i++ {
		pp := path.Join(fmt.Sprintf("sect%d", i), fmt.Sprintf("page%d.md", i))
		page, _ := s.getPageNew(nil, pp)
		c.Assert(page, qt.Not(qt.IsNil), qt.Commentf(pp))
	}
}

func BenchmarkGetPageRegular(b *testing.B) {
	var (
		c = qt.New(b)
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	)

	s := createGetPageRegularBenchmarkSite(b)

	pagePaths := make([]string, b.N)

	for i := 0; i < b.N; i++ {
		pagePaths[i] = path.Join(fmt.Sprintf("sect%d", r.Intn(10)), fmt.Sprintf("page%d.md", r.Intn(100)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		page, _ := s.getPageNew(nil, pagePaths[i])
		c.Assert(page, qt.Not(qt.IsNil))
	}
}

type getPageTest struct {
	kind          string
	context       page.Page
	pathVariants  []string
	expectedTitle string
}

func (t *getPageTest) check(p page.Page, err error, errorMsg string, c *qt.C) {
	c.Helper()
	errorComment := qt.Commentf(errorMsg)
	switch t.kind {
	case "Ambiguous":
		c.Assert(err, qt.Not(qt.IsNil))
		c.Assert(p, qt.IsNil, errorComment)
	case "NoPage":
		c.Assert(err, qt.IsNil)
		c.Assert(p, qt.IsNil, errorComment)
	default:
		c.Assert(err, qt.IsNil, errorComment)
		c.Assert(p, qt.Not(qt.IsNil), errorComment)
		c.Assert(p.Kind(), qt.Equals, t.kind, errorComment)
		c.Assert(p.Title(), qt.Equals, t.expectedTitle, errorComment)
	}
}

func TestGetPage(t *testing.T) {

	var (
		cfg, fs = newTestCfg()
		c       = qt.New(t)
	)

	pc := func(title string) string {
		return fmt.Sprintf(pageCollectionsPageTemplate, title)
	}

	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			content := pc(fmt.Sprintf("Title%d_%d", i, j))
			writeSource(t, fs, filepath.Join("content", fmt.Sprintf("sect%d", i), fmt.Sprintf("page%d.md", j)), content)
		}
	}

	content := pc("home page")
	writeSource(t, fs, filepath.Join("content", "_index.md"), content)

	content = pc("about page")
	writeSource(t, fs, filepath.Join("content", "about.md"), content)

	content = pc("section 3")
	writeSource(t, fs, filepath.Join("content", "sect3", "_index.md"), content)

	writeSource(t, fs, filepath.Join("content", "sect3", "unique.md"), pc("UniqueBase"))
	writeSource(t, fs, filepath.Join("content", "sect3", "Unique2.md"), pc("UniqueBase2"))

	content = pc("another sect7")
	writeSource(t, fs, filepath.Join("content", "sect3", "sect7", "_index.md"), content)

	content = pc("deep page")
	writeSource(t, fs, filepath.Join("content", "sect3", "subsect", "deep.md"), content)

	// Bundle variants
	writeSource(t, fs, filepath.Join("content", "sect3", "b1", "index.md"), pc("b1 bundle"))
	writeSource(t, fs, filepath.Join("content", "sect3", "index", "index.md"), pc("index bundle"))

	s := buildSingleSite(t, deps.DepsCfg{Fs: fs, Cfg: cfg}, BuildCfg{SkipRender: true})

	sec3, err := s.getPageNew(nil, "/sect3")
	c.Assert(err, qt.IsNil)
	c.Assert(sec3, qt.Not(qt.IsNil))

	tests := []getPageTest{
		// legacy content root relative paths
		{page.KindHome, nil, []string{""}, "home page"},
		{page.KindPage, nil, []string{"about.md", "ABOUT.md"}, "about page"},
		{page.KindSection, nil, []string{"sect3"}, "section 3"},
		{page.KindPage, nil, []string{"sect3/page1.md"}, "Title3_1"},
		{page.KindPage, nil, []string{"sect4/page2.md"}, "Title4_2"},
		{page.KindSection, nil, []string{"sect3/sect7"}, "another sect7"},
		{page.KindPage, nil, []string{"sect3/subsect/deep.md"}, "deep page"},
		{page.KindPage, nil, []string{filepath.FromSlash("sect5/page3.md")}, "Title5_3"}, //test OS-specific path

		// shorthand refs (potentially ambiguous)
		{page.KindPage, nil, []string{"unique.md", "unique"}, "UniqueBase"},
		{page.KindPage, nil, []string{"Unique2.md", "unique2.md", "unique2"}, "UniqueBase2"},
		{"Ambiguous", nil, []string{"page1.md"}, ""},

		// ISSUE: This is an ambiguous ref, but because we have to support the legacy
		// content root relative paths without a leading slash, the lookup
		// returns /sect7. This undermines ambiguity detection, but we have no choice.
		//{"Ambiguous", nil, []string{"sect7"}, ""},
		{page.KindSection, nil, []string{"sect7"}, "Sect7s"},

		// absolute paths
		{page.KindHome, nil, []string{"/", ""}, "home page"},
		{page.KindPage, nil, []string{"/about.md", "/about"}, "about page"},
		{page.KindSection, nil, []string{"/sect3"}, "section 3"},
		{page.KindPage, nil, []string{"/sect3/page1.md", "/Sect3/Page1.md"}, "Title3_1"},
		{page.KindPage, nil, []string{"/sect4/page2.md"}, "Title4_2"},
		{page.KindSection, nil, []string{"/sect3/sect7"}, "another sect7"},
		{page.KindPage, nil, []string{"/sect3/subsect/deep.md"}, "deep page"},
		{page.KindPage, nil, []string{filepath.FromSlash("/sect5/page3.md")}, "Title5_3"}, //test OS-specific path
		{page.KindPage, nil, []string{"/sect3/unique.md"}, "UniqueBase"},
		{page.KindPage, nil, []string{"/sect3/Unique2.md", "/sect3/unique2.md", "/sect3/unique2", "/sect3/Unique2"}, "UniqueBase2"},
		//next test depends on this page existing
		// {"NoPage", nil, []string{"/unique.md"}, ""},  // ISSUE #4969: this is resolving to /sect3/unique.md
		{"NoPage", nil, []string{"/missing-page.md"}, ""},
		{"NoPage", nil, []string{"/missing-section"}, ""},

		// relative paths
		{page.KindHome, sec3, []string{".."}, "home page"},
		{page.KindHome, sec3, []string{"../"}, "home page"},
		{page.KindPage, sec3, []string{"../about.md"}, "about page"},
		{page.KindSection, sec3, []string{"."}, "section 3"},
		{page.KindSection, sec3, []string{"./"}, "section 3"},
		{page.KindPage, sec3, []string{"page1.md"}, "Title3_1"},
		{page.KindPage, sec3, []string{"./page1.md"}, "Title3_1"},
		{page.KindPage, sec3, []string{"../sect4/page2.md"}, "Title4_2"},
		{page.KindSection, sec3, []string{"sect7"}, "another sect7"},
		{page.KindSection, sec3, []string{"./sect7"}, "another sect7"},
		{page.KindPage, sec3, []string{"./subsect/deep.md"}, "deep page"},
		{page.KindPage, sec3, []string{"./subsect/../../sect7/page9.md"}, "Title7_9"},
		{page.KindPage, sec3, []string{filepath.FromSlash("../sect5/page3.md")}, "Title5_3"}, //test OS-specific path
		{page.KindPage, sec3, []string{"./unique.md"}, "UniqueBase"},
		{"NoPage", sec3, []string{"./sect2"}, ""},
		//{"NoPage", sec3, []string{"sect2"}, ""}, // ISSUE: /sect3 page relative query is resolving to /sect2

		// absolute paths ignore context
		{page.KindHome, sec3, []string{"/"}, "home page"},
		{page.KindPage, sec3, []string{"/about.md"}, "about page"},
		{page.KindPage, sec3, []string{"/sect4/page2.md"}, "Title4_2"},
		{page.KindPage, sec3, []string{"/sect3/subsect/deep.md"}, "deep page"}, //next test depends on this page existing
		{"NoPage", sec3, []string{"/subsect/deep.md"}, ""},

		// Taxonomies
		{page.KindTaxonomyTerm, nil, []string{"categories"}, "Categories"},
		{page.KindTaxonomy, nil, []string{"categories/hugo", "categories/Hugo"}, "Hugo"},

		// Bundle variants
		{page.KindPage, nil, []string{"sect3/b1", "sect3/b1/index.md", "sect3/b1/index.en.md"}, "b1 bundle"},
		{page.KindPage, nil, []string{"sect3/index/index.md", "sect3/index"}, "index bundle"},
	}

	for i, test := range tests {
		c.Run(fmt.Sprintf("t%d", i+1), func(c *qt.C) {
			errorMsg := fmt.Sprintf("Test case %v %v -> %s", test.context, test.pathVariants, test.expectedTitle)

			// test legacy public Site.GetPage (which does not support page context relative queries)
			if test.context == nil {
				for _, ref := range test.pathVariants {
					args := append([]string{test.kind}, ref)
					page, err := s.Info.GetPage(args...)
					test.check(page, err, errorMsg, c)
				}
			}

			// test new internal Site.getPageNew
			for _, ref := range test.pathVariants {
				page2, err := s.getPageNew(test.context, ref)
				test.check(page2, err, errorMsg, c)
			}

		})
	}

}
