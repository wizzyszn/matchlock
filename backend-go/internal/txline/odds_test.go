package txline

import "testing"

func TestParse1X2Odds(t *testing.T) {
	rows := []OddsPayload{
		{
			SuperOddsType: "ASIANHANDICAP_PARTICIPANT_GOALS",
			PriceNames:    []string{"part1", "part2"},
			Prices:        []int32{2000, 1900},
			Ts:            1,
		},
		{
			SuperOddsType:    "1X2_PARTICIPANT_RESULT",
			MarketParameters: "",
			MarketPeriod:     "",
			PriceNames:       []string{"part1", "draw", "part2"},
			Prices:           []int32{1349, 5765, 11700},
			Ts:               2,
		},
	}

	odds, ok := Parse1X2Odds(rows)
	if !ok {
		t.Fatal("expected odds")
	}
	if odds.Home != 1.349 || odds.Draw != 5.765 || odds.Away != 11.7 {
		t.Fatalf("odds = %#v", odds)
	}
}

func TestParse1X2OddsMissing(t *testing.T) {
	if _, ok := Parse1X2Odds(nil); ok {
		t.Fatal("expected false")
	}
}