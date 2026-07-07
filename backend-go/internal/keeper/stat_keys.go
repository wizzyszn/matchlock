package keeper

import chainsol "github.com/matchlock/backend-go/internal/solana"

// TxLINE soccer full-time outcome stat keys (period 4).
// See https://txline.txodds.com/documentation/examples/onchain-validation
const (
	statKeyDraw       uint32 = 1001
	statKeyP1Win      uint32 = 1002
	statKeyP2Win      uint32 = 1003
)

// StatKeyForWinningSide maps a Matchlock outcome to the TxLINE stat key that
// should read value=1 with predicate (threshold=0, greaterThan) at the final seq.
func StatKeyForWinningSide(side uint8, participant1IsHome bool) uint32 {
	switch side {
	case chainsol.SideDraw:
		return statKeyDraw
	case chainsol.SideHome:
		if participant1IsHome {
			return statKeyP1Win
		}
		return statKeyP2Win
	case chainsol.SideAway:
		if participant1IsHome {
			return statKeyP2Win
		}
		return statKeyP1Win
	default:
		return statKeyP1Win
	}
}