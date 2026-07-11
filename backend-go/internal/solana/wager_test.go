package solana

import (
	"testing"

	"github.com/gagliardetto/solana-go"
)

func TestDecodeWagerAndHelpers(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	maker := solana.NewWallet().PublicKey()
	taker := solana.NewWallet().PublicKey()
	matchID := "17952170"

	data := make([]byte, wagerAccountSizeV2)
	copy(data[:8], wagerDiscriminator[:])
	copy(data[8:40], maker.Bytes())
	copy(data[72:104], taker.Bytes())
	copy(data[matchIDOffsetV2:matchIDOffsetV2+32], MatchIDFilterBytes(matchID))
	data[136] = uint8(len(matchID))
	data[137] = SideAway
	data[138] = SideHome
	data[139] = 0x40
	data[140] = 0x42
	data[141] = 0x0f
	data[142] = 0x00
	data[143] = 0x00
	data[144] = 0x00
	data[145] = 0x00
	data[146] = 0x00
	data[147] = WagerStatusMatched
	data[148] = 1
	data[149] = 2

	wager, err := DecodeWager(pubkey, data)
	if err != nil {
		t.Fatalf("DecodeWager: %v", err)
	}
	if wager.MatchIDString() != matchID {
		t.Fatalf("match_id = %q", wager.MatchIDString())
	}
	if wager.Stake != 0x000f4240 {
		t.Fatalf("stake = %d", wager.Stake)
	}
	if StatusName(wager.Status) != "matched" {
		t.Fatalf("status = %q", StatusName(wager.Status))
	}
	if SideName(wager.MakerSide) != "away" {
		t.Fatalf("side = %q", SideName(wager.MakerSide))
	}

	winner, err := wager.WinnerPubkey(SideAway)
	if err != nil || !winner.Equals(maker) {
		t.Fatalf("winner = %v err=%v", winner, err)
	}
}

func TestDecodeWagerV3ParticipantOrientation(t *testing.T) {
	pubkey := solana.NewWallet().PublicKey()
	matchID := "17952170"
	data := make([]byte, wagerAccountSizeV3)
	copy(data[:8], wagerDiscriminator[:])
	copy(data[8:40], solana.NewWallet().PublicKey().Bytes())
	copy(data[72:104], solana.NewWallet().PublicKey().Bytes())
	copy(data[matchIDOffsetV3:matchIDOffsetV3+32], MatchIDFilterBytes(matchID))
	data[136] = uint8(len(matchID))
	data[participant1IsHomeOffsetV3] = 0
	data[makerSideOffsetV3] = SideHome
	data[takerSideOffsetV3] = SideAway
	data[stakeOffsetV3] = 1
	data[statusOffsetV3] = WagerStatusMatched
	data[149] = 1
	data[150] = 2

	wager, err := DecodeWager(pubkey, data)
	if err != nil {
		t.Fatalf("DecodeWager: %v", err)
	}
	if wager.Participant1IsHome {
		t.Fatal("expected participant1_is_home=false")
	}
	if wager.MakerSide != SideHome || wager.TakerSide != SideAway || wager.Stake != 1 {
		t.Fatalf("decoded wager = %#v", wager)
	}
}

func TestDecodeWagerRejectsInvalidDiscriminator(t *testing.T) {
	_, err := DecodeWager(solana.NewWallet().PublicKey(), make([]byte, wagerAccountSizeV2))
	if err == nil {
		t.Fatal("expected discriminator error")
	}
}

func TestWagerHelpers(t *testing.T) {
	if StatusName(WagerStatusOpen) != "open" ||
		StatusName(WagerStatusMatched) != "matched" ||
		StatusName(WagerStatusSettled) != "settled" ||
		StatusName(WagerStatusCancelled) != "cancelled" ||
		StatusName(99) != "unknown" {
		t.Fatal("StatusName mismatch")
	}
	if SideName(SideHome) != "home" ||
		SideName(SideAway) != "away" ||
		SideName(SideDraw) != "draw" {
		t.Fatal("SideName mismatch")
	}

	wager := Wager{
		MakerSide: SideHome,
		TakerSide: SideAway,
		Maker:     solana.NewWallet().PublicKey(),
		Taker:     solana.NewWallet().PublicKey(),
	}
	if _, err := wager.WinnerPubkey(9); err == nil {
		t.Fatal("expected winner error")
	}
	winner, err := wager.WinnerPubkey(SideAway)
	if err != nil || !winner.Equals(wager.Taker) {
		t.Fatalf("taker winner = %v err=%v", winner, err)
	}

	_, err = DecodeWager(solana.NewWallet().PublicKey(), []byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected size error")
	}
}
