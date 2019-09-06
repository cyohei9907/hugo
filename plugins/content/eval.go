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
	"io"

	"github.com/spf13/cast"

	"github.com/gohugoio/hugo/helpers"
	"go.starlark.net/starlark"

	"github.com/starlight-go/starlight"
	"github.com/starlight-go/starlight/convert"
)

type sourcePluginFiles struct {
	FilenamesProvider
	SourcePlugin
}

type sourcePluginStream struct {
	StreamProvider
	SourcePlugin
}

type sourcePluginFunc func(item map[string]interface{}) Bundle

func (f sourcePluginFunc) ToBundle(item map[string]interface{}) Bundle {
	return f(item)
}

type getFilenamesFunc func() []string

func (f getFilenamesFunc) GetFilenames() []string {
	return f()
}

var globals = make(map[string]interface{})
var thread = &starlark.Thread{} // TODO1
// TODO1 cache

func EvalSourcePlugin(r io.Reader) (SourcePlugin, error) {
	out, err := starlight.Eval(helpers.ReaderToBytes(r), globals, nil)
	if err != nil {
		return nil, err
	}

	return toSourcePlugin(out)
}

func toSourcePlugin(out map[string]interface{}) (SourcePlugin, error) {
	if fn, found := out["GetFilenames"]; found {
		gf := getFilenamesFunc(
			func() []string {
				return cast.ToStringSlice(starlarkCall(thread, fn))
			},
		)

		return sourcePluginFiles{
			FilenamesProvider: gf,
			SourcePlugin:      getToBundleFunc(out),
		}, nil
	}

	return nil, nil
}

func getToBundleFunc(out map[string]interface{}) sourcePluginFunc {
	if fn, found := out["ToBundle"]; found {
		return func(item map[string]interface{}) Bundle {
			return starlarkCall(thread, fn, item).(Bundle)
		}
	}

	return func(item map[string]interface{}) Bundle {
		return Bundle{}
	}
}

func starlarkCall(thread *starlark.Thread, fn interface{}, args ...interface{}) interface{} {
	argsv := make(starlark.Tuple, len(args))
	for i, arg := range args {
		argv, err := convert.ToValue(arg)
		if err != nil {
			panic(err) // TODO1
		}
		argsv[i] = argv
	}
	v, err := starlark.Call(thread, fn.(*starlark.Function), argsv, nil)
	if err != nil {
		panic(err)
	}

	return fromValue(v)
}

func fromValue(v starlark.Value) interface{} {
	switch v := v.(type) {
	default:
		return convert.FromValue(v)
	}
}
