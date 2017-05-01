// Copyright 2017 The Hugo Authors. All rights reserved.
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

package time

import (
	"github.com/spf13/hugo/deps"
	"github.com/spf13/hugo/tpl/internal"
)

const name = "time"

func init() {
	f := func(d *deps.Deps) *internal.TemplateFuncsNamespace {
		ctx := New()

		ns := &internal.TemplateFuncsNamespace{
			Name:    name,
			Context: func() interface{} { return ctx },
		}

		ns.AddMethodMapping(ctx.Format,
			[]string{"dateFormat"},
			[][2]string{
				{`dateFormat: {{ dateFormat "Monday, Jan 2, 2006" "2015-01-21" }}`, `dateFormat: Wednesday, Jan 21, 2015`},
			},
		)

		ns.AddMethodMapping(ctx.Now,
			[]string{"now"},
			[][2]string{},
		)

		ns.AddMethodMapping(ctx.AsTime,
			[]string{"asTime"}, // TODO(bep) handle duplicate
			[][2]string{
				{`{{ (asTime "2015-01-21").Year }}`, `2015`},
			},
		)

		return ns

	}

	internal.AddTemplateFuncsNamespace(f)
}
