package service

// LotteryPrizeCandidate is deliberately small so the draw rule can be tested
// without a database or random source.
type LotteryPrizeCandidate struct {
	ID      uint
	Stock   int
	WinRate int
}

// ChooseLotteryPrize maps a number in [1, 100] to the configured prize bands.
// Any unused percentage is an explicit "谢谢参与" result.
func ChooseLotteryPrize(items []LotteryPrizeCandidate, roll int) (LotteryPrizeCandidate, bool) {
	if roll < 1 || roll > 100 {
		return LotteryPrizeCandidate{}, false
	}
	upperBound := 0
	for _, item := range items {
		if item.Stock <= 0 || item.WinRate <= 0 {
			continue
		}
		upperBound += item.WinRate
		if roll <= upperBound {
			return item, true
		}
	}
	return LotteryPrizeCandidate{}, false
}
