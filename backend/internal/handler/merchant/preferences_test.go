package merchant

import "testing"

func TestNormalizeGenerationPreferencesDefaultsAndDedupes(t *testing.T) {
	prefs, err := normalizeGenerationPreferenceRequest(generationPreferenceRequest{
		FocusKeywords:    []string{" 香辣蟹 ", "服务热情", "香辣蟹", ""},
		StyleCodes:       []string{},
		ReferenceReviews: []string{"  蟹很入味，服务员会主动帮忙换盘。  "},
		LengthVariance:   "",
	})
	if err != nil {
		t.Fatalf("normalizeGenerationPreferenceRequest returned error: %v", err)
	}
	if got := prefs.FocusKeywords; len(got) != 2 || got[0] != "香辣蟹" || got[1] != "服务热情" {
		t.Fatalf("focus keywords got %#v", got)
	}
	if got := prefs.StyleCodes; len(got) != 1 || got[0] != "natural" {
		t.Fatalf("style codes got %#v, want default natural", got)
	}
	if got := prefs.ReferenceReviews; len(got) != 1 || got[0] != "蟹很入味，服务员会主动帮忙换盘。" {
		t.Fatalf("reference reviews got %#v", got)
	}
	if prefs.LengthVariance != "wide" {
		t.Fatalf("length variance got %q, want wide", prefs.LengthVariance)
	}
}

func TestNormalizeGenerationPreferencesRejectsTooManyAndTooLong(t *testing.T) {
	tooManyKeywords := generationPreferenceRequest{
		FocusKeywords: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
		StyleCodes:    []string{"natural"},
	}
	if _, err := normalizeGenerationPreferenceRequest(tooManyKeywords); err == nil {
		t.Fatal("expected too many focus keywords to be rejected")
	}

	longReview := generationPreferenceRequest{
		StyleCodes:       []string{"natural"},
		ReferenceReviews: []string{repeatRune('好', 301)},
	}
	if _, err := normalizeGenerationPreferenceRequest(longReview); err == nil {
		t.Fatal("expected long reference review to be rejected")
	}

	unknownStyle := generationPreferenceRequest{
		StyleCodes: []string{"salesy"},
	}
	if _, err := normalizeGenerationPreferenceRequest(unknownStyle); err == nil {
		t.Fatal("expected unknown style code to be rejected")
	}
}

func repeatRune(r rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
