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

	// V2 offsets (with invited_taker)
	statusOffsetV2     = 147
	matchIDOffsetV2    = 104
	takerSideOffsetV2  = 138
	invitedTakerOffsetV2 = 40

	// V1 offsets (no invited_taker)
	statusOffsetV1    = 115
	matchIDOffsetV1   = 72
	takerSideOffsetV1 = 106
)

// Wager is a decoded on-chain wager account.
type Wager struct {
	Pubkey        solana.PublicKey
	Maker         solana.PublicKey
	InvitedTaker  solana.PublicKey
	Taker         solana.PublicKey
	MatchID   [32]byte
	MatchIDLen uint8
	MakerSide uint8
	TakerSide uint8
	Stake     uint64
	Status    uint8
	Bump      uint8
	VaultBump uint8
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

	if len(data) == wagerAccountSizeV1 {
		// V1 Layout: no invited_taker
		w.Taker = solana.PublicKeyFromBytes(data[40:72])
		copy(w.MatchID[:], data[matchIDOffsetV1:matchIDOffsetV1+32])
		w.MatchIDLen = data[104]
		w.MakerSide = data[105]
		w.TakerSide = data[takerSideOffsetV1]
		w.Stake = uint64(data[107]) | uint64(data[108])<<8 | uint64(data[109])<<16 | uint64(data[110])<<24 |
			uint64(data[111])<<32 | uint64(data[112])<<40 | uint64(data[113])<<48 | uint64(data[114])<<56
		w.Status = data[statusOffsetV1]
		w.Bump = data[116]
		w.VaultBump = data[117]
	} else {
		// V2 Layout: has invited_taker
		w.InvitedTaker = solana.PublicKeyFromBytes(data[invitedTakerOffsetV2 : invitedTakerOffsetV2+32])
		w.Taker = solana.PublicKeyFromBytes(data[72:104])
		copy(w.MatchID[:], data[matchIDOffsetV2:matchIDOffsetV2+32])
		w.MatchIDLen = data[136]
		w.MakerSide = data[137]
		w.TakerSide = data[takerSideOffsetV2]
		w.Stake = uint64(data[139]) | uint64(data[140])<<8 | uint64(data[141])<<16 | uint64(data[142])<<24 |
			uint64(data[143])<<32 | uint64(data[144])<<40 | uint64(data[145])<<48 | uint64(data[146])<<56
		w.Status = data[statusOffsetV2]
		w.Bump = data[148]
		w.VaultBump = data[149]
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