package translator

import (
	"gplover/dictionary"
	"gplover/stroke"
	"strings"
)

// Translation represents a successful dictionary lookup for one or more strokes.
type Translation struct {
	Strokes  []stroke.Stroke
	Rtfcre   []string
	English  string
	Replaced []Translation
}

// State holds the translation history.
type State struct {
	Translations []Translation
}

// Translator is the main engine for converting strokes to translations.
type Translator struct {
	Dict       dictionary.Dictionary
	State      State
	UndoBuffer []Translation
	Listeners  []func([]Translation, []Translation, *Translation)
	MaxHistory int
}

// NewTranslator creates a new Translator instance.
func NewTranslator(dict dictionary.Dictionary, maxHistory int) *Translator {
	return &Translator{
		Dict:       dict,
		State:      State{},
		MaxHistory: maxHistory,
	}
}

// Translate adds a new stroke and emits the appropriate translations.
func (tr *Translator) Translate(s *stroke.Stroke) Translation {
	// Try greedy match from history + this stroke
	allStrokes := collectStrokes(tr.State.Translations)
	allStrokes = append(allStrokes, *s)

	maxLen := 42
	for size := min(len(allStrokes), maxLen); size >= 1; size-- {
		start := len(allStrokes) - size
		chunk := allStrokes[start:]
		rtfcre := stroke.ToRtfcre(chunk)
		if eng, ok := tr.Dict.Lookup(strings.Join(rtfcre, "")); ok {
			t := Translation{
				Strokes:  chunk,
				Rtfcre:   rtfcre,
				English:  eng,
				Replaced: tr.findReplaced(size),
			}
			tr.applyTranslation(t)
			return t
		}
	}

	// Fallback: untranslated stroke
	t := Translation{
		Strokes: []stroke.Stroke{*s},
		Rtfcre:  stroke.ToRtfcre([]stroke.Stroke{*s}),
	}
	tr.applyTranslation(t)
	return t
}

func (tr *Translator) applyTranslation(t Translation) {
	// Remove replaced entries from history
	tr.State.Translations = tr.State.Translations[:len(tr.State.Translations)-len(t.Replaced)]
	tr.State.Translations = append(tr.State.Translations, t)
}

func (tr *Translator) findReplaced(strokeCount int) []Translation {
	var replaced []Translation
	total := 0
	for i := len(tr.State.Translations) - 1; i >= 0 && total < strokeCount; i-- {
		t := tr.State.Translations[i]
		total += len(t.Strokes)
		replaced = append([]Translation{t}, replaced...)
	}
	return replaced
}

func (tr *Translator) UndoLast() []Translation {
	if len(tr.State.Translations) == 0 {
		return nil
	}
	last := tr.State.Translations[len(tr.State.Translations)-1]
	tr.State.Translations = tr.State.Translations[:len(tr.State.Translations)-1]
	if len(last.Replaced) > 0 {
		tr.State.Translations = append(tr.State.Translations, last.Replaced...)
	}
	return []Translation{last}
}

func collectStrokes(ts []Translation) []stroke.Stroke {
	var strokes []stroke.Stroke
	for _, t := range ts {
		strokes = append(strokes, t.Strokes...)
	}
	return strokes
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
