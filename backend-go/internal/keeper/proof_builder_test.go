package keeper

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

func TestProofBuilderRejectsInferredFinal(t *testing.T) {
	store := newMemCache()
	const matchID = "17952170"
	home, away := int32(2), int32(1)
	store.matches[matchID] = cache.Match{
		MatchID:     matchID,
		FixtureID:   17952170,
		GameState:   "FT",
		IsFinal:     true,
		FinalSource: cache.FinalSourceInferred,
		HomeGoals:   &home,
		AwayGoals:   &away,
		Seq:         100,
		UpdatedAt:   time.Now().UTC(),
	}

	wager := chainsol.Wager{
		Pubkey:             solana.NewWallet().PublicKey(),
		Maker:              solana.NewWallet().PublicKey(),
		Taker:              solana.NewWallet().PublicKey(),
		MatchIDLen:         uint8(len(matchID)),
		Participant1IsHome: true,
		MakerSide:          chainsol.SideHome,
		TakerSide:          chainsol.SideAway,
		Stake:              1_000_000,
		Status:             chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte(matchID))

	_, err := (&ProofBuilder{Cache: store}).BuildForWager(context.Background(), wager)
	if err == nil {
		t.Fatal("expected inferred final to be rejected")
	}
	if !strings.Contains(err.Error(), "not verified") {
		t.Fatalf("err = %q, want not verified", err.Error())
	}
}

type proofProgramLookup struct {
	program solana.PublicKey
}

func (p proofProgramLookup) TxlineProgramID() solana.PublicKey {
	return p.program
}

func TestProofBuilderRefreshesInferredFinal(t *testing.T) {
	store := newMemCache()
	const matchID = "17952170"
	home, away := int32(0), int32(0)
	store.matches[matchID] = cache.Match{
		MatchID:     matchID,
		FixtureID:   17952170,
		GameState:   "FT",
		IsFinal:     true,
		FinalSource: cache.FinalSourceInferred,
		HomeGoals:   &home,
		AwayGoals:   &away,
		Seq:         1,
		UpdatedAt:   time.Now().UTC(),
	}

	wager := chainsol.Wager{
		Pubkey:             solana.NewWallet().PublicKey(),
		Maker:              solana.NewWallet().PublicKey(),
		Taker:              solana.NewWallet().PublicKey(),
		MatchIDLen:         uint8(len(matchID)),
		Participant1IsHome: true,
		MakerSide:          chainsol.SideHome,
		TakerSide:          chainsol.SideAway,
		Stake:              1_000_000,
		Status:             chainsol.WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte(matchID))

	tx := &snapshotTxline{rows: []txline.ScoreSnapshotRow{{
		FixtureID:          17952170,
		GameState:          "F2",
		Seq:                42,
		Participant1IsHome: true,
		ScoreSoccer: &txline.SoccerFixtureScore{
			Participant1: txline.SoccerTotalScore{Goals: 2},
			Participant2: txline.SoccerTotalScore{Goals: 0},
		},
	}}}
	proof, err := (&ProofBuilder{
		Cache:  store,
		Txline: tx,
		Solana: proofProgramLookup{program: solana.NewWallet().PublicKey()},
	}).BuildForWager(context.Background(), wager)
	if err != nil {
		t.Fatalf("BuildForWager: %v", err)
	}
	if proof.WinningSide != chainsol.SideHome || proof.StatKey != statKeyP1Win {
		t.Fatalf("proof = %#v", proof)
	}
	got, err := store.GetMatch(context.Background(), matchID)
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if got.FinalSource != cache.FinalSourceTxline || got.Seq != 42 {
		t.Fatalf("refreshed match = %#v", got)
	}
}
