package tokenestimate

import "testing"

func TestEstimateTextCountsCJKRunesConservatively(t *testing.T) {
	text := repeatRune('春', 100)

	if got, want := EstimateText(text), 100; got != want {
		t.Fatalf("EstimateText(CJK) = %d, want %d", got, want)
	}
}

func TestEstimateTextCompactsASCIIWordRuns(t *testing.T) {
	text := repeatRune('a', 100)

	if got, want := EstimateText(text), 25; got != want {
		t.Fatalf("EstimateText(ASCII run) = %d, want %d", got, want)
	}
}

func TestEstimateTextHandlesMixedRunsAndPunctuation(t *testing.T) {
	text := "hello, 春天!"

	if got, want := EstimateText(text), 7; got != want {
		t.Fatalf("EstimateText(mixed) = %d, want %d", got, want)
	}
}

func TestEstimateTextCountsJSONPunctuationIndividually(t *testing.T) {
	text := `{"x":1,"y":2}`

	if got, want := EstimateText(text), 13; got != want {
		t.Fatalf("EstimateText(JSON punctuation) = %d, want %d", got, want)
	}
}

func TestEstimateTextCountsEmojiAndFullWidthPunctuationAsRunes(t *testing.T) {
	text := "🙂，。"

	if got, want := EstimateText(text), 3; got != want {
		t.Fatalf("EstimateText(emoji/full-width punctuation) = %d, want %d", got, want)
	}
}

func TestEstimateStableJSONIsDeterministicForMapOrder(t *testing.T) {
	left := map[string]any{"b": 2, "a": "春"}
	right := map[string]any{"a": "春", "b": 2}

	leftJSON, err := CompactStableJSON(left)
	if err != nil {
		t.Fatalf("CompactStableJSON(left) error = %v", err)
	}
	rightJSON, err := CompactStableJSON(right)
	if err != nil {
		t.Fatalf("CompactStableJSON(right) error = %v", err)
	}
	if leftJSON != rightJSON {
		t.Fatalf("CompactStableJSON differs by map insertion order:\nleft=%s\nright=%s", leftJSON, rightJSON)
	}

	leftTokens, err := EstimateStableJSON(left)
	if err != nil {
		t.Fatalf("EstimateStableJSON(left) error = %v", err)
	}
	rightTokens, err := EstimateStableJSON(right)
	if err != nil {
		t.Fatalf("EstimateStableJSON(right) error = %v", err)
	}
	if leftTokens != rightTokens {
		t.Fatalf("EstimateStableJSON differs by map insertion order: left=%d right=%d", leftTokens, rightTokens)
	}
}

func TestEstimateStableJSONIncreasesWithAdditionalContent(t *testing.T) {
	base, err := EstimateStableJSON(map[string]any{"text": "春"})
	if err != nil {
		t.Fatalf("EstimateStableJSON(base) error = %v", err)
	}
	larger, err := EstimateStableJSON(map[string]any{"text": "春天"})
	if err != nil {
		t.Fatalf("EstimateStableJSON(larger) error = %v", err)
	}
	if larger <= base {
		t.Fatalf("larger estimate = %d, want greater than base %d", larger, base)
	}
}

func repeatRune(r rune, count int) string {
	out := make([]rune, count)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
