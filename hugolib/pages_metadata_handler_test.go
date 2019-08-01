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
	"bytes"
	"fmt"
	"testing"

	"github.com/gohugoio/hugo/parser"
	"github.com/gohugoio/hugo/parser/metadecoders"
)

func TestCascade(t *testing.T) {

	weight := 0

	p := func(m map[string]interface{}) string {
		var yamlStr string

		if len(m) > 0 {
			var b bytes.Buffer

			parser.InterfaceToConfig(m, metadecoders.YAML, &b)
			yamlStr = b.String()
		}

		weight += 10

		metaStr := "---\n" + yamlStr + fmt.Sprintf("\nweight: %d\n---", weight)

		return metaStr

	}

	b := newTestSitesBuilder(t).WithConfigFile("toml", `
baseURL = "https://example.org"
disableKinds = ["taxonomy", "taxonomyTerm"]
`)
	b.WithContent(
		"_index.md", p(map[string]interface{}{
			"title": "Home",
			"cascade": map[string]interface{}{
				"title": "Cascade Home",
				"icon":  "home.png",
			},
		}),
		"p1.md", p(map[string]interface{}{
			"title": "p1",
		}),
		"p2.md", p(map[string]interface{}{}),
		"sect1/_index.md", p(map[string]interface{}{
			"title": "Sect1",
			"cascade": map[string]interface{}{
				"title": "Cascade Sect1",
				"icon":  "sect1.png",
			},
		}),
		"sect1/s1_2/_index.md", p(map[string]interface{}{
			"title": "Sect1_2",
		}),
		"sect1/s1_2/p1.md", p(map[string]interface{}{
			"title": "Sect1_2_p1",
		}),
		"sect2/_index.md", p(map[string]interface{}{
			"title": "Sect2",
		}),
		"sect2/p1.md", p(map[string]interface{}{
			"title": "Sect2_p1",
		}),
		"sect2/p2.md", p(map[string]interface{}{}),
	)

	b.WithTemplates("index.html", `
	
{{ range .Site.Pages }}
{{- .Weight }}|{{ .Kind }}|{{ .Path }}|{{ .Title }}|{{ .Params.icon }}|
{{ end }}
`)

	b.Build(BuildCfg{})

	b.AssertFileContent("public/index.html", `10|home|_index.md|Home|home.png|
20|page|p1.md|p1|home.png|
30|page|p2.md|Cascade Home|home.png|
40|section|sect1/_index.md|Sect1|sect1.png|
50|section|sect1/s1_2/_index.md|Sect1_2|sect1.png|
60|page|sect1/s1_2/p1.md|Sect1_2_p1|sect1.png|
70|section|sect2/_index.md|Sect2|home.png|
80|page|sect2/p1.md|Sect2_p1|home.png|
90|page|sect2/p2.md|Cascade Home|home.png|`)
}
