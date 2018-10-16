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

package hugolib

type pageContent struct {
	rawFrontMatter []byte

	// rawContent is the raw content read from the content file.
	rawContent []byte

	// workContent is a copy of rawContent that may be mutated during site build.
	workContent []byte
}

// TODO(bep) errors rename contentBytes contentWithoutFrontMatterBytes
func (c pageContent) rawContentWithoutFrontMatter() []byte {
	return c.rawContent
}

func (c pageContent) frontMatterBytes() []byte {
	return c.rawFrontMatter
}
