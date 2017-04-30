// Copyright 2016 The Hugo Authors. All rights reserved.
//
// Portions Copyright The Go Authors.

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

package tplimpl

import (
	"fmt"
	"html/template"
	"sync"

	"github.com/spf13/cast"
	"github.com/spf13/hugo/tpl/internal"

	// Init the namespaces
	_ "github.com/spf13/hugo/tpl/compare"
	_ "github.com/spf13/hugo/tpl/data"
	_ "github.com/spf13/hugo/tpl/lang"
	_ "github.com/spf13/hugo/tpl/math"
	_ "github.com/spf13/hugo/tpl/strings"
)

// Get retrieves partial output from the cache based upon the partial name.
// If the partial is not found in the cache, the partial is rendered and added
// to the cache.
func (t *templateFuncster) Get(key, name string, context interface{}) (p interface{}, err error) {
	var ok bool

	t.cachedPartials.RLock()
	p, ok = t.cachedPartials.p[key]
	t.cachedPartials.RUnlock()

	if ok {
		return
	}

	t.cachedPartials.Lock()
	if p, ok = t.cachedPartials.p[key]; !ok {
		t.cachedPartials.Unlock()
		p, err = t.partial(name, context)

		t.cachedPartials.Lock()
		t.cachedPartials.p[key] = p

	}
	t.cachedPartials.Unlock()

	return
}

// partialCache represents a cache of partials protected by a mutex.
type partialCache struct {
	sync.RWMutex
	p map[string]interface{}
}

// partialCached executes and caches partial templates.  An optional variant
// string parameter (a string slice actually, but be only use a variadic
// argument to make it optional) can be passed so that a given partial can have
// multiple uses.  The cache is created with name+variant as the key.
func (t *templateFuncster) partialCached(name string, context interface{}, variant ...string) (interface{}, error) {
	key := name
	if len(variant) > 0 {
		for i := 0; i < len(variant); i++ {
			key += variant[i]
		}
	}
	return t.Get(key, name, context)
}

func (t *templateFuncster) initFuncMap() {
	funcMap := template.FuncMap{
		// Namespaces
		"collections": t.collections.Namespace,
		"crypto":      t.crypto.Namespace,
		"encoding":    t.encoding.Namespace,
		"images":      t.images.Namespace,
		"inflect":     t.inflect.Namespace,
		"os":          t.os.Namespace,
		"safe":        t.safe.Namespace,
		//"time":        t.time.Namespace,
		"transform": t.transform.Namespace,
		"urls":      t.urls.Namespace,

		"absURL":        t.urls.AbsURL,
		"absLangURL":    t.urls.AbsLangURL,
		"after":         t.collections.After,
		"apply":         t.collections.Apply,
		"base64Decode":  t.encoding.Base64Decode,
		"base64Encode":  t.encoding.Base64Encode,
		"dateFormat":    t.time.Format,
		"delimit":       t.collections.Delimit,
		"dict":          t.collections.Dictionary,
		"echoParam":     t.collections.EchoParam,
		"emojify":       t.transform.Emojify,
		"first":         t.collections.First,
		"getenv":        t.os.Getenv,
		"highlight":     t.transform.Highlight,
		"htmlEscape":    t.transform.HTMLEscape,
		"htmlUnescape":  t.transform.HTMLUnescape,
		"humanize":      t.inflect.Humanize,
		"imageConfig":   t.images.Config,
		"in":            t.collections.In,
		"index":         t.collections.Index,
		"int":           func(v interface{}) (int, error) { return cast.ToIntE(v) },
		"intersect":     t.collections.Intersect,
		"isSet":         t.collections.IsSet,
		"isset":         t.collections.IsSet,
		"jsonify":       t.encoding.Jsonify,
		"last":          t.collections.Last,
		"markdownify":   t.transform.Markdownify,
		"md5":           t.crypto.MD5,
		"now":           t.time.Now,
		"partial":       t.partial,
		"partialCached": t.partialCached,
		"plainify":      t.transform.Plainify,
		"pluralize":     t.inflect.Pluralize,
		"print":         fmt.Sprint,
		"printf":        fmt.Sprintf,
		"println":       fmt.Sprintln,
		"querify":       t.collections.Querify,
		"readDir":       t.os.ReadDir,
		"readFile":      t.os.ReadFile,
		"ref":           t.urls.Ref,
		"relURL":        t.urls.RelURL,
		"relLangURL":    t.urls.RelLangURL,
		"relref":        t.urls.RelRef,
		"safeCSS":       t.safe.CSS,
		"safeHTML":      t.safe.HTML,
		"safeHTMLAttr":  t.safe.HTMLAttr,
		"safeJS":        t.safe.JS,
		"safeJSStr":     t.safe.JSStr,
		"safeURL":       t.safe.URL,
		"sanitizeURL":   t.safe.SanitizeURL,
		"sanitizeurl":   t.safe.SanitizeURL,
		"seq":           t.collections.Seq,
		"sha1":          t.crypto.SHA1,
		"sha256":        t.crypto.SHA256,
		"shuffle":       t.collections.Shuffle,
		"singularize":   t.inflect.Singularize,
		"slice":         t.collections.Slice,
		"sort":          t.collections.Sort,
		"string":        func(v interface{}) (string, error) { return cast.ToStringE(v) },
		"time":          t.time.AsTime,
		"union":         t.collections.Union,
		"urlize":        t.PathSpec.URLize,
		"where":         t.collections.Where,
	}

	// Merge the namespace funcs
	for _, nsf := range internal.TemplateFuncsNamespaceRegistry {
		ns := nsf(t.Deps)
		// TODO(bep) namespace ns.Context is a dummy func just to make this work.
		// Consider if we can add this context to the rendering context in an easy
		// way to make this cleaner. Maybe.
		funcMap[ns.Name] = ns.Context
		for k, v := range ns.Aliases {
			funcMap[k] = v
		}
	}

	t.funcMap = funcMap
	t.Tmpl.(*templateHandler).setFuncs(funcMap)
	t.collections.Funcs(funcMap)
}
