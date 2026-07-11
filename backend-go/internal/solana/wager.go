package solana

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
)

var wagerDiscriminator = [8]byte{3, 110, 53, 190, 113, 31, 230, 40}

// SystemProgramID is the default/unset counterparty pubkey on Open wagers.
var SystemProgramID = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

// HasCounterparty reports whether the wager has a real matched opponent.
func (w Wager) HasCounterparty() bool {
	if w.Status == WagerStatusOpen {
		return false
	}
	return !w.Taker.IsZero() && !w.Taker.Equals(SystemProgramID)
}

const (
	WagerStatusOpen      uint8 = 0
	WagerStatusMatched   uint8 = 1
	WagerStatusSettled   uint8 = 2
	WagerStatusCancelled uint8 = 3

	SideHome uint8 = 0
	SideAway uint8 = 1
	SideDraw uint8 = 2

	wagerAccountSizeV1 = 118
	wagerAccountSizeV2 = 150
	wagerAccountSizeV3 = 151
	wagerAccountSizeV4 = 159

	// V4 offsets (with nonce)
	statusOffsetV4             = 148
	matchIDOffsetV4            = 104
	participant1IsHomeOffsetV4 = 137
	makerSideOffsetV4          = 138
	takerSideOffsetV4          = 139
	stakeOffsetV4              = 140
	nonceOffsetV4              = 149
	bumpOffsetV4               = 157
	vaultBumpOffsetV4          = 158

	// V3 offsets (with invited_taker and participant1_is_home)
	statusOffsetV3             = 148
	matchIDOffsetV3            = 104
	participant1IsHomeOffsetV3 = 137
	makerSideOffsetV3          = 138
	takerSideOffsetV3          = 139
	stakeOffsetV3              = 140

	// V2 offsets (with invited_taker)
	statusOffsetV2       = 147
	matchIDOffsetV2      = 104
	makerSideOffsetV2    = 137
	takerSideOffsetV2    = 138
	stakeOffsetV2        = 139
	invitedTakerOffsetV2 = 40

	// V1 offsets (no invited_taker)
	statusOffsetV1    = 115
	matchIDOffsetV1   = 72
	makerSideOffsetV1 = 105
	takerSideOffsetV1 = 106
	stakeOffsetV1     = 107
)

// Wager is a decoded on-chain wager account.
type Wager struct {
	Pubkey             solana.PublicKey
	Maker              solana.PublicKey
	InvitedTaker       solana.PublicKey
	Taker              solana.PublicKey
	MatchID            [32]byte
	MatchIDLen         uint8
	Participant1IsHome bool
	MakerSide          uint8
	TakerSide          uint8
	Stake              uint64
	Status             uint8
	Nonce              uint64
	Bump               uint8
	VaultBump          uint8
}

func DecodeWager(pubkey solana.PublicKey, data []byte) (Wager, error) {
	if len(data) < wagerAccountSizeV1 {
		return Wager{}, fmt.Errorf("wager account too small: %d", len(data))
	}
	for i := range wagerDiscriminator {
		if data[i] != wagerDiscriminator[i] {
			return Wager{}, fmt.Errorf("invalid wager discriminator")
		}
	}

	w := Wager{
		Pubkey: pubkey,
		Maker:  solana.PublicKeyFromBytes(data[8:40]),
	}

	switch len(data) {
	case wagerAccountSizeV1:
		// V1 Layout: no invited_taker
		w.Taker = solana.PublicKeyFromBytes(data[40:72])
		copy(w.MatchID[:], data[matchIDOffsetV1:matchIDOffsetV1+32])
		w.MatchIDLen = data[104]
		w.Participant1IsHome = true
		w.MakerSide = data[makerSideOffsetV1]
		w.TakerSide = data[takerSideOffsetV1]
		w.Stake = uint64(data[stakeOffsetV1]) | uint64(data[stakeOffsetV1+1])<<8 | uint64(data[stakeOffsetV1+2])<<16 | uint64(data[stakeOffsetV1+3])<<24 |
			uint64(data[stakeOffsetV1+4])<<32 | uint64(data[stakeOffsetV1+5])<<40 | uint64(data[stakeOffsetV1+6])<<48 | uint64(data[stakeOffsetV1+7])<<56
		w.Status = data[statusOffsetV1]
		w.Bump = data[116]
		w.VaultBump = data[117]
	case wagerAccountSizeV2:
		// V2 Layout: has invited_taker
		w.InvitedTaker = solana.PublicKeyFromBytes(data[invitedTakerOffsetV2 : invitedTakerOffsetV2+32])
		w.Taker = solana.PublicKeyFromBytes(data[72:104])
		copy(w.MatchID[:], data[matchIDOffsetV2:matchIDOffsetV2+32])
		w.MatchIDLen = data[136]
		w.Participant1IsHome = true
		w.MakerSide = data[makerSideOffsetV2]
		w.TakerSide = data[takerSideOffsetV2]
		w.Stake = uint64(data[stakeOffsetV2]) | uint64(data[stakeOffsetV2+1])<<8 | uint64(data[stakeOffsetV2+2])<<16 | uint64(data[stakeOffsetV2+3])<<24 |
			uint64(data[stakeOffsetV2+4])<<32 | uint64(data[stakeOffsetV2+5])<<40 | uint64(data[stakeOffsetV2+6])<<48 | uint64(data[stakeOffsetV2+7])<<56
		w.Status = data[statusOffsetV2]
		w.Bump = data[148]
		w.VaultBump = data[149]
	case wagerAccountSizeV3:
		// V3 Layout: V2 plus TxLINE participant orientation
		w.InvitedTaker = solana.PublicKeyFromBytes(data[invitedTakerOffsetV2 : invitedTakerOffsetV2+32])
		w.Taker = solana.PublicKeyFromBytes(data[72:104])
		copy(w.MatchID[:], data[matchIDOffsetV3:matchIDOffsetV3+32])
		w.MatchIDLen = data[136]
		w.Participant1IsHome = data[participant1IsHomeOffsetV3] != 0
		w.MakerSide = data[makerSideOffsetV3]
		w.TakerSide = data[takerSideOffsetV3]
		w.Stake = uint64(data[stakeOffsetV3]) | uint64(data[stakeOffsetV3+1])<<8 | uint64(data[stakeOffsetV3+2])<<16 | uint64(data[stakeOffsetV3+3])<<24 |
			uint64(data[stakeOffsetV3+4])<<32 | uint64(data[stakeOffsetV3+5])<<40 | uint64(data[stakeOffsetV3+6])<<48 | uint64(data[stakeOffsetV3+7])<<56
		w.Status = data[statusOffsetV3]
		w.Bump = data[149]
		w.VaultBump = data[150]
	case wagerAccountSizeV4:
		// V4 Layout: V3 plus nonce
		w.InvitedTaker = solana.PublicKeyFromBytes(data[invitedTakerOffsetV2 : invitedTakerOffsetV2+32])
		w.Taker = solana.PublicKeyFromBytes(data[72:104])
		copy(w.MatchID[:], data[matchIDOffsetV4:matchIDOffsetV4+32])
		w.MatchIDLen = data[136]
		w.Participant1IsHome = data[participant1IsHomeOffsetV4] != 0
		w.MakerSide = data[makerSideOffsetV4]
		w.TakerSide = data[takerSideOffsetV4]
		w.Stake = uint64(data[stakeOffsetV4]) | uint64(data[stakeOffsetV4+1])<<8 | uint64(data[stakeOffsetV4+2])<<16 | uint64(data[stakeOffsetV4+3])<<24 |
			uint64(data[stakeOffsetV4+4])<<32 | uint64(data[stakeOffsetV4+5])<<40 | uint64(data[stakeOffsetV4+6])<<48 | uint64(data[stakeOffsetV4+7])<<56
		w.Status = data[statusOffsetV4]
		w.Nonce = uint64(data[nonceOffsetV4]) | uint64(data[nonceOffsetV4+1])<<8 | uint64(data[nonceOffsetV4+2])<<16 | uint64(data[nonceOffsetV4+3])<<24 |
			uint64(data[nonceOffsetV4+4])<<32 | uint64(data[nonceOffsetV4+5])<<40 | uint64(data[nonceOffsetV4+6])<<48 | uint64(data[nonceOffsetV4+7])<<56
		w.Bump = data[bumpOffsetV4]
		w.VaultBump = data[vaultBumpOffsetV4]
	default:
		return Wager{}, fmt.Errorf("unsupported wager account size: %d", len(data))
	}

	return w, nil
}

func (w Wager) MatchIDString() string {
	return string(w.MatchID[:w.MatchIDLen])
}

func (w Wager) WinnerPubkey(winningSide uint8) (solana.PublicKey, error) {
	switch winningSide {
	case w.MakerSide:
		return w.Maker, nil
	case w.TakerSide:
		return w.Taker, nil
	default:
		return solana.PublicKey{}, fmt.Errorf("winning side does not match maker/taker")
	}
}

func MatchIDFilterBytes(matchID string) []byte {
	b := make([]byte, 32)
	copy(b, []byte(matchID))
	return b
}

// StatusName returns a stable API label for a wager status byte.
func StatusName(status uint8) string {
	switch status {
	case WagerStatusOpen:
		return "open"
	case WagerStatusMatched:
		return "matched"
	case WagerStatusSettled:
		return "settled"
	case WagerStatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// SideName returns home/draw/away for API consumers.
func SideName(side uint8) string {
	switch side {
	case SideAway:
		return "away"
	case SideDraw:
		return "draw"
	default:
		return "home"
	}
}
