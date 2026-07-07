package solana

import (
	"encoding/base64"
	"testing"

	"github.com/matchlock/backend-go/internal/txline"
)

func TestValidationFromAPIWithProofNodes(t *testing.T) {
	hash := txline.FlexHash(base64.StdEncoding.EncodeToString(make([]byte, 32)))
	v := txline.StatValidation{
		Summary: txline.StatValidationSummary{
			FixtureID:             1,
			EventStatsSubTreeRoot: hash,
			UpdateStats: txline.ScoresUpdateStatsResp{
				UpdateCount: 1, MinTimestamp: 1, MaxTimestamp: 2,
			},
		},
		EventStatRoot: hash,
		SubTreeProof: []txline.ProofNodeResponse{{
			Hash: hash, IsRightSibling: true,
		}},
		MainTreeProof: []txline.ProofNodeResponse{{
			Hash: hash, IsRightSibling: false,
		}},
		StatToProve: txline.ScoreStatResponse{Key: 1, Value: 1, Period: 0},
		StatProof: []txline.ProofNodeResponse{{
			Hash: hash, IsRightSibling: true,
		}},
	}
	args, _, err := ValidationFromAPI(v)
	if err != nil {
		t.Fatalf("ValidationFromAPI: %v", err)
	}
	if len(args.FixtureProof) != 1 || len(args.MainTreeProof) != 1 {
		t.Fatalf("proofs = %d %d", len(args.FixtureProof), len(args.MainTreeProof))
	}
	data, err := EncodeSettleWagerData(args, SideHome, [32]byte{1})
	if err != nil || len(data) < 20 {
		t.Fatalf("encode: len=%d err=%v", len(data), err)
	}
}