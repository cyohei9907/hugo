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
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestPagesFromYAML(t *testing.T) {
	b := newTestSitesBuilder(t)

	b.WithContent("page.md", "")

	b.WithSourceFile("assets/data/content1.yaml", `
title: Yaml Page 1
---
title: Yaml Page 2
`)

	b.WithSourceFile("assets/data/content2.yaml", `
title: Yaml Page 3
---
title: Yaml Page 4
`)

	b.WithContent("_content.py", `

def GetFilenames():
	return ["data/content1.yaml", "data/content2.yaml"]
	
`)

	b.Build(BuildCfg{})

	s := b.H.Sites[0]

	b.Assert(s.RegularPages(), qt.HasLen, 4)
}
