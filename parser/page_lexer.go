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
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

const eof = -1

type (
	itemType int
	pos      int
)

const (
	tError itemType = iota
	tEOF

	tHTMLStart

	tDelimYAML
	tDelimJSON
	tDelimTOML

	tFrontMatter

	tText
)

type item struct {
	typ itemType
	pos pos
	val []byte
}

type pagelexer struct {
	input []byte
	state stateFunc

	pos     pos // input position
	start   pos // item start position
	width   pos // width of last element
	lastPos pos // position of the last item returned by nextItem

	items []item
}

func (i item) String() string {
	switch {
	case i.typ == tEOF:
		return "EOF"
	default:
		return string(i.val)
	}
}

func (l *pagelexer) run() {
	for l.state = l.outside(); l.state != nil; {
		l.state = l.state(l)
	}
}

type stateFunc func(*pagelexer) stateFunc

func (l *pagelexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}

	runeValue, runeWidth := utf8.DecodeRune(l.input[l.pos:])
	l.width = pos(runeWidth)
	l.pos += l.width
	return runeValue
}

func (l *pagelexer) emit(t itemType) {
	l.items = append(l.items, item{t, l.start, l.input[l.start:l.pos]})
	l.start = l.pos
}

func (l *pagelexer) ignore() {
	l.start = l.pos
}

// nil terminates the parser
func (l *pagelexer) errorf(format string, args ...interface{}) stateFunc {
	l.items = append(l.items, item{tError, l.start, []byte(fmt.Sprintf(format, args...))})
	return nil
}

func (l *pagelexer) lineNum(pos pos) int {
	return bytes.Count(l.input[:pos], []byte("\n")) + 1
}

func (l *pagelexer) outside() stateFunc {
LOOP:
	for {

		// If first non space == "<" => not renderable => break
		// TODO(bep) errors check the BOM etc. rules
		switch r := l.next(); {
		case unicode.IsSpace(r):
			l.ignore()
		case r == '-':
			if bytes.HasPrefix(l.input[l.pos:], []byte("--")) {
				l.pos += 3
				l.emit(tDelimYAML)
				return l.frontMatter(tDelimYAML)
			}
			fallthrough
		case r == '+':
			if bytes.HasPrefix(l.input[l.pos:], []byte("++")) {
				l.pos += 3
				l.emit(tDelimTOML)
				return l.frontMatter(tDelimTOML)
			}
			fallthrough
		case r == '{':
		case r == '<':
			// This is a HTML document, no need to read further.
			l.emit(tHTMLStart)
			fallthrough
		default:
			break LOOP

		}
	}
	// Done!
	if l.pos > l.start {
		l.emit(tText)
	}
	l.emit(tEOF)
	return nil
}

func (l *pagelexer) frontMatter(t itemType) stateFunc {
	delim := []byte("---\n")
	if t == tDelimTOML {
		delim = []byte("+++\n")
	}

	end := bytes.Index(l.input[l.pos:], delim)

	if end == -1 {
		return l.errorf("end front matter delim %q not found", delim)
	}

	l.pos += pos(end)
	l.emit(tText)
	l.pos += 4
	l.emit(t)
	l.pos = pos(len(l.input))
	l.emit(tText)
	l.emit(tEOF)
	return nil
}
