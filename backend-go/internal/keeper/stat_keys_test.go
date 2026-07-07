package keeper

import (
	"testing"

	chainsol "github.com/matchlock/backend-go/internal/solana"
)

func TestStatKeyForWinningSide(t *testing.T) {
	cases := []struct {
		side      uint8
		p1Home    bool
		want      uint32
	}{
		{chainsol.SideDraw, true, statKeyDraw},
		{chainsol.SideHome, true, statKeyP1Win},
		{chainsol.SideAway, true, statKeyP2Win},
		{chainsol.SideHome, false, statKeyP2Win},
		{chainsol.SideAway, false, statKeyP1Win},
	}
	for _, tc := range cases {
		if got := StatKeyForWinningSide(tc.side, tc.p1Home); got != tc.want {
			t.Fatalf("side=%d p1Home=%v got=%d want=%d", tc.side, tc.p1Home, got, tc.want)
		}
	}
}