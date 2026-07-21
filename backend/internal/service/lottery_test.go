package service

import "testing"

func TestChooseLotteryPrizeUsesConfiguredRatesAndIgnoresEmptyStock(t *testing.T) {
	prize, won := ChooseLotteryPrize([]LotteryPrizeCandidate{
		{ID: 1, Stock: 0, WinRate: 100},
		{ID: 2, Stock: 3, WinRate: 25},
	}, 10)
	if !won || prize.ID != 2 {
		t.Fatalf("expected the available prize to win, got prize=%+v won=%v", prize, won)
	}
}

func TestChooseLotteryPrizeReturnsNoPrizeOutsideConfiguredRates(t *testing.T) {
	prize, won := ChooseLotteryPrize([]LotteryPrizeCandidate{{ID: 2, Stock: 3, WinRate: 25}}, 26)
	if won || prize.ID != 0 {
		t.Fatalf("expected no prize outside the configured rate, got prize=%+v won=%v", prize, won)
	}
}
