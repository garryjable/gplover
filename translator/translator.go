// Copyright (c) 2025 Garrett result.
// This File is part of sten. Sten is free software under GPLv3 .
// See LICENSE.txt for details.

package translator

import (
	"sten/dictionary"
	"strings"
)

type Translation struct {
	result    string
	stroke    string
	prev      *Translation // previous
	multiPrev *Translation // for multistroke translations
}

// Translator is the main engine for converting strokes to translations.
type Translator struct {
	dict          dictionary.Dict
	latest        *Translation
	maxOutlineLen int
	in            chan string
	out           chan *Translation
}

func newCommand(result string, stroke string) *Translation {
	return &Translation{
		result:    result,
		stroke:    stroke,
		prev:      nil,
		multiPrev: nil,
	}
}

func newWord(result string, stroke string, prev *Translation) *Translation {
	return &Translation{
		result:    result,
		stroke:    stroke,
		prev:      prev,
		multiPrev: nil,
	}
}

func newMultiWord(result string, stroke string, prev *Translation, multiPrev *Translation) *Translation {
	return &Translation{
		result:    result,
		stroke:    stroke,
		prev:      prev,
		multiPrev: multiPrev,
	}
}

func newUntranslatable(stroke string, prev *Translation) *Translation {
	return &Translation{
		result:    "",
		stroke:    stroke,
		prev:      prev,
		multiPrev: nil,
	}
}

// NewTranslator creates a new Translator instance.
func NewTranslator(dict dictionary.Dict, maxOutlineLen int) *Translator {
	t := &Translator{
		dict: dict,
		latest: &Translation{
			result:    "",
			stroke:    "",
			prev:      nil,
			multiPrev: nil,
		},
		maxOutlineLen: maxOutlineLen,
		in:            make(chan string, 16),
		out:           make(chan *Translation, 16),
	}
	go t.run()
	return t
}

func (tr *Translation) PrintHistory() {
	if tr.prev != nil {
		tr.prev.PrintHistory()
	}
}

func (tr *Translator) getLatest(stroke string, outline string, prev *Translation, strokeCount int) *Translation {
	if strokeCount <= tr.maxOutlineLen {
		if prev.prev != nil {
			latest := tr.getLatest(stroke, prev.stroke+"/"+outline, prev.prev, strokeCount+1)
			if latest != nil {
				return latest // return the longest possible match
			}
		}
		if result, ok := tr.dict.Lookup(outline); ok {
			if strings.HasPrefix(result, "=") {
				return newCommand(result, stroke)
			} else if strokeCount == 1 {
				return newWord(result, stroke, tr.latest)
			} else {
				return newMultiWord(result, stroke, tr.latest, prev)
			}
		} else if strokeCount == 1 {
			return newUntranslatable(stroke, tr.latest)
		}
	}
	return nil // dont seek longer than possible matches
}

func (t *Translation) Text() string {
	if t.result != "" {
		return t.result
	} else {
		return t.stroke
	}

}

func (t *Translation) isCommand() bool {
	if strings.HasPrefix(t.result, "=") {
		return true
	} else {
		return false
	}

}

func (tr *Translator) appendHistory(latest *Translation) {
	if !latest.isCommand() {
		tr.latest = latest
	}
}

// For engine to send strokes:
func (t *Translator) Translate(stroke string) {
	t.in <- stroke
}

// For closing (when done, eg: engine detects machine done)
func (t *Translator) Close() {
	close(t.in)
}

func (t *Translator) Out() <-chan *Translation {
	return t.out
}

func (t *Translator) run() {
	for stroke := range t.in {
		latest := t.getLatest(stroke, stroke, t.latest, 1)
		t.appendHistory(latest)
		t.out <- latest
	}
	close(t.out)
}
