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

package parser

import (
	"reflect"
	"testing"
)

type pageLexerTest struct {
	name  string
	input string
	items []item
}

func TestPageLexer(t *testing.T) {

	var (
		itemYAMLDelim = item{tDelimYAML, 0, []byte("---\n")}
		itemTOMLDelim = item{tDelimYAML, 0, []byte("+++\n")}

		itemEOF             = item{tEOF, 0, []byte("")}
		itemFrontMatterYAML = item{tText, 0, []byte("title: \"foo\"\n")}
		itemFrontMatterTOML = item{tText, 0, []byte("title = \"foo\"\n")}
		//itemFrontMatterJSON = "{\n   \"title\": \"test 1\"\n}\n"
		itemContent = item{tText, 0, []byte("Content.\n\nSome more content.\n")}
	)

	var pageLexerTests = []pageLexerTest{
		{"empty", "", []item{itemEOF}},
		{"Basic YAML", `

   
---
title: "foo"
---
Content.

Some more content.
`, []item{itemYAMLDelim, itemFrontMatterYAML, itemYAMLDelim, itemContent, itemEOF}},
		{"Basic TOML", `
+++
title = "foo"
+++
Content.

Some more content.
`, []item{itemTOMLDelim, itemFrontMatterTOML, itemTOMLDelim, itemContent, itemEOF}},
	}

	for _, test := range pageLexerTests {
		lexer := &pagelexer{
			input: []byte(test.input),
		}
		lexer.run()
		if !equal(test.items, lexer.items) {
			t.Fatalf("[%s] expected:\n%s\ngot:\n%s", test.name, test.items, lexer.items)
		}

	}

}

func equal(i1, i2 []item) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if !reflect.DeepEqual(i1[k].val, i2[k].val) {
			return false
		}
	}
	return true
}
