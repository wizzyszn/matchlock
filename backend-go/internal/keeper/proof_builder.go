package keeper

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/cache"
	chainsol "github.com/matchlock/backend-go/internal/solana"
	"github.com/matchlock/backend-go/internal/txline"
)

// SettlementProof is the payload a winner wallet needs to submit settle_wager.
type SettlementProof struct {
	WinningSide     uint8                 `json:"winning_side_code"`
	WinningSideName string                `json:"winning_side"`
	FixtureID       int64                 `json:"fixture_id"`
	Seq             int32                 `json:"seq"`
	StatKey         uint32                `json:"stat_key"`
	Validation      txline.StatValidation `json:"validation"`
	MerkleRoot      string                `json:"merkle_root"`
	EpochDay        uint16                `json:"epoch_day"`
	DailyScoresPDA  string                `json:"daily_scores_pda"`
	TxlineProgramID string                `json:"txline_program_id"`
}

// TxlineProgramLookup supplies the TxLINE program id for PDA derivation.
type TxlineProgramLookup interface {
	TxlineProgramID() solana.PublicKey
}

// ProofBuilder resolves TxLINE proofs for permissionless winner settlement.
type ProofBuilder struct {
	Cache  cache.Store
	Txline TxlineClient
	Solana TxlineProgramLookup
}

// BuildForWager returns settlement proof data when the match has a TxLINE-verified final score.
func (b *ProofBuilder) BuildForWager(ctx context.Context, wager chainsol.Wager) (SettlementProof, error) {
	if wager.Status != chainsol.WagerStatusMatched {
		return SettlementProof{}, fmt.Errorf("wager status is not matched")
	}

	matchID := wager.MatchIDString()
	match, err := b.Cache.GetMatch(ctx, matchID)
	if err != nil {
		return SettlementProof{}, fmt.Errorf("load match %s: %w", matchID, err)
	}
	if !match.IsFinal {
		return SettlementProof{}, fmt.Errorf("match %s is not final", matchID)
	}

	worker := &Worker{Cache: b.Cache, Txline: b.Txline}
	refreshed, update, err := worker.RefreshVerifiedFinal(ctx, match)
	if err != nil {
		return SettlementProof{}, fmt.Errorf("match %s final score is not verified: %w", matchID, err)
	}
	if refreshed.FinalSource != cache.FinalSourceTxline {
		return SettlementProof{}, fmt.Errorf("match %s final score is not verified", matchID)
	}

	winningSide, ok := winningSideFromScore(update)
	if !ok {
		return SettlementProof{}, fmt.Errorf("cannot determine winning side for match %s", matchID)
	}

	validation, statKey, err := worker.fetchDeclaredWinStatValidation(ctx, update.FixtureID, update.Seq, winningSide, wager.Participant1IsHome)
	if err != nil {
		return SettlementProof{}, err
	}

	args, merkleRoot, err := chainsol.ValidationFromAPI(validation)
	if err != nil {
		return SettlementProof{}, fmt.Errorf("map validation: %w", err)
	}

	epochDay := chainsol.EpochDayFromMillis(args.TS)
	dailyScores, _, err := chainsol.FindDailyScoresRootsPDA(b.Solana.TxlineProgramID(), epochDay)
	if err != nil {
		return SettlementProof{}, err
	}

	return SettlementProof{
		WinningSide:     winningSide,
		WinningSideName: chainsol.SideName(winningSide),
		FixtureID:       update.FixtureID,
		Seq:             update.Seq,
		StatKey:         statKey,
		Validation:      validation,
		MerkleRoot:      base64.StdEncoding.EncodeToString(merkleRoot[:]),
		EpochDay:        epochDay,
		DailyScoresPDA:  dailyScores.String(),
		TxlineProgramID: b.Solana.TxlineProgramID().String(),
	}, nil
}
