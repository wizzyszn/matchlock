package solana

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/matchlock/backend-go/internal/txline"
)

func TestDecodeHash32Encodings(t *testing.T) {
	zeros := make([]byte, 32)
	b64 := base64.StdEncoding.EncodeToString(zeros)
	if _, err := decodeHash32(b64); err != nil {
		t.Fatalf("base64: %v", err)
	}

	hex := strings.Repeat("ab", 32)
	got, err := decodeHash32(hex)
	if err != nil {
		t.Fatalf("hex: %v", err)
	}
	if got[0] != 0xab {
		t.Fatalf("byte = %#x", got[0])
	}

	if _, err := decodeHash32(""); err == nil {
		t.Fatal("expected empty hash error")
	}
	rawB64 := base64.RawStdEncoding.EncodeToString(zeros)
	if _, err := decodeHash32(rawB64); err != nil {
		t.Fatalf("raw base64: %v", err)
	}
	if _, err := decodeHash32("not-a-hash"); err == nil {
		t.Fatal("expected unsupported encoding error")
	}
}

func TestDecodeHex(t *testing.T) {
	raw, err := decodeHex("0102ff")
	if err != nil || len(raw) != 3 || raw[2] != 0xff {
		t.Fatalf("raw = %#v err=%v", raw, err)
	}
	if _, err := decodeHex("0"); err == nil {
		t.Fatal("expected odd length error")
	}
	if _, err := decodeHex("zz"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestProofNodesFromAPISkipsInvalid(t *testing.T) {
	nodes := proofNodesFromAPI([]txline.ProofNodeResponse{
		{Hash: txline.FlexHash("not-valid")},
		{Hash: txline.FlexHash(base64.StdEncoding.EncodeToString(make([]byte, 32))), IsRightSibling: true},
	})
	if len(nodes) != 1 || !nodes[0].IsRightSibling {
		t.Fatalf("nodes = %#v", nodes)
	}
}

func TestValidationFromAPIErrors(t *testing.T) {
	if _, _, err := ValidationFromAPI(txline.StatValidation{}); err == nil {
		t.Fatal("expected root error")
	}
	v := txline.StatValidation{
		Summary: txline.StatValidationSummary{
			EventStatsSubTreeRoot: txline.FlexHash(base64.StdEncoding.EncodeToString(make([]byte, 32))),
		},
	}
	if _, _, err := ValidationFromAPI(v); err == nil {
		t.Fatal("expected event stat root error")
	}
}

func TestEncodeSettleWagerDataWithOptionalFields(t *testing.T) {
	statB := StatTerm{
		StatToProve: ScoreStat{Key: 2, Value: 1, Period: 0},
	}
	op := uint8(1)
	args := ValidateStatArgs{
		TS: 1700000000000,
		FixtureSummary: ScoresBatchSummary{
			FixtureID: 17952170,
			UpdateStats: ScoresUpdateStats{
				UpdateCount:  1,
				MinTimestamp: 1700000000000,
				MaxTimestamp: 1700000000000,
			},
		},
		FixtureProof: []ProofNode{{IsRightSibling: true}},
		MainTreeProof: []ProofNode{{
			Hash:           [32]byte{1},
			IsRightSibling: false,
		}},
		Predicate: TraderPredicate{Threshold: 1, Comparison: 1},
		StatA: StatTerm{
			StatToProve:   ScoreStat{Key: 1002, Value: 1, Period: 0},
			EventStatRoot: [32]byte{2},
			StatProof:     []ProofNode{{Hash: [32]byte{3}, IsRightSibling: true}},
		},
		StatB: &statB,
		Op:    &op,
	}
	data, err := EncodeSettleWagerData(args, SideHome, [32]byte{9})
	if err != nil {
		t.Fatalf("EncodeSettleWagerData: %v", err)
	}
	if len(data) < 16 {
		t.Fatalf("data too short: %d", len(data))
	}
}