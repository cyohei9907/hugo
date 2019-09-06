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

package content

import (
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestEvalSourcePlugin(t *testing.T) {
	c := qt.New(t)

	pluginFiles := `

def GetFilenames():
	return ["file1.json", "file2.json"]

`

	plugin, err := EvalSourcePlugin(strings.NewReader(pluginFiles))
	c.Assert(err, qt.IsNil)
	source, ok := plugin.(SourcePluginFiles)
	c.Assert(ok, qt.Equals, true)
	c.Assert(source.GetFilenames(), qt.DeepEquals, []string{"file1.json", "file2.json"})

}
