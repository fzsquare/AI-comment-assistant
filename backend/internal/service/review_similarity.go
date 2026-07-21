package service

import (
	"math"
	"strings"
	"unicode"
)

const (
	ReviewMatchAlgorithmVersion = "review-text-v1"
	minLooseMatchRunes          = 12
	minContainmentRunes         = 18
	minCommonSubstringRunes     = 24
	minCommonSubstringRatio     = 0.70
	minCharacterSimilarity      = 0.86
)

type ReviewTextMatchResult struct {
	Matched          bool
	Score            float64
	Reason           string
	AlgorithmVersion string
}

func CompareReviewText(left string, right string) ReviewTextMatchResult {
	a := NormalizeReviewText(left)
	b := NormalizeReviewText(right)
	result := ReviewTextMatchResult{AlgorithmVersion: ReviewMatchAlgorithmVersion}
	if a == "" || b == "" {
		return result
	}
	if a == b {
		result.Matched = true
		result.Score = 1
		result.Reason = "exact"
		return result
	}

	shorter := minRuneLen(a, b)
	if shorter < minLooseMatchRunes {
		return result
	}

	if shorter >= minContainmentRunes && (strings.Contains(a, b) || strings.Contains(b, a)) {
		result.Matched = true
		result.Score = containmentScore(a, b)
		result.Reason = "contains"
		return result
	}

	common := longestCommonSubstringRunes(a, b)
	if common >= minCommonSubstringRunes || float64(common)/float64(shorter) >= minCommonSubstringRatio {
		result.Matched = true
		result.Score = math.Max(float64(common)/float64(maxRuneLen(a, b)), minCharacterSimilarity)
		result.Reason = "common_substring"
		return result
	}

	score := jaccardRuneSimilarity(a, b)
	if score >= minCharacterSimilarity {
		result.Matched = true
		result.Score = score
		result.Reason = "character_similarity"
	}
	return result
}

func NormalizeReviewText(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		builder.WriteRune(widthFoldRune(r))
	}
	return builder.String()
}

func widthFoldRune(r rune) rune {
	if r == 0x3000 {
		return ' '
	}
	if r >= 0xff01 && r <= 0xff5e {
		return r - 0xfee0
	}
	return r
}

func minRuneLen(a string, b string) int {
	la := len([]rune(a))
	lb := len([]rune(b))
	if la < lb {
		return la
	}
	return lb
}

func maxRuneLen(a string, b string) int {
	la := len([]rune(a))
	lb := len([]rune(b))
	if la > lb {
		return la
	}
	return lb
}

func containmentScore(a string, b string) float64 {
	shorter := minRuneLen(a, b)
	longer := maxRuneLen(a, b)
	if longer == 0 {
		return 0
	}
	return float64(shorter) / float64(longer)
}

func longestCommonSubstringRunes(a string, b string) int {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 || len(br) == 0 {
		return 0
	}
	prev := make([]int, len(br)+1)
	best := 0
	for i := 1; i <= len(ar); i++ {
		curr := make([]int, len(br)+1)
		for j := 1; j <= len(br); j++ {
			if ar[i-1] == br[j-1] {
				curr[j] = prev[j-1] + 1
				if curr[j] > best {
					best = curr[j]
				}
			}
		}
		prev = curr
	}
	return best
}

func jaccardRuneSimilarity(a string, b string) float64 {
	left := runeCounts(a)
	right := runeCounts(b)
	if len(left) == 0 || len(right) == 0 {
		return 0
	}

	var intersection int
	var union int
	seen := map[rune]struct{}{}
	for r, lc := range left {
		rc := right[r]
		if lc < rc {
			intersection += lc
		} else {
			intersection += rc
		}
		if lc > rc {
			union += lc
		} else {
			union += rc
		}
		seen[r] = struct{}{}
	}
	for r, rc := range right {
		if _, ok := seen[r]; ok {
			continue
		}
		union += rc
	}
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func runeCounts(value string) map[rune]int {
	counts := map[rune]int{}
	for _, r := range value {
		counts[r]++
	}
	return counts
}
