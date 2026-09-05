package tokenestimate

import (
	"encoding/json"
	"strings"
)

// EstimateText returns a deterministic provider-neutral token estimate.
func EstimateText(value string) int {
	total := 0
	asciiRunes := 0
	flushASCII := func() {
		if asciiRunes == 0 {
			return
		}
		total += (asciiRunes + 3) / 4
		asciiRunes = 0
	}

	for _, r := range value {
		if isASCIIWordLike(r) {
			asciiRunes++
			continue
		}
		flushASCII()
		total++
	}
	flushASCII()
	return total
}

// CompactStableJSON returns compact JSON with Go's deterministic map-key order.
func CompactStableJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// EstimateStableJSON estimates the compact JSON representation of value.
func EstimateStableJSON(value any) (int, error) {
	data, err := CompactStableJSON(value)
	if err != nil {
		return 0, err
	}
	return EstimateText(data), nil
}

// EstimateJSONDocument estimates a parsed JSON document after stable compaction.
func EstimateJSONDocument(document string) (int, error) {
	decoder := json.NewDecoder(strings.NewReader(document))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return 0, err
	}
	return EstimateStableJSON(value)
}

func isASCIIWordLike(r rune) bool {
	return r == ' ' ||
		r == '\n' ||
		r == '\r' ||
		r == '\t' ||
		(r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}
