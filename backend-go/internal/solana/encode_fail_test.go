package solana

import (
	"errors"
	"testing"
)

type errWriter struct {
	failOn int
	writes int
}

func (w *errWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes >= w.failOn {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

func TestWriteHelpersPropagateErrors(t *testing.T) {
	root := [32]byte{1}
	args := ValidateStatArgs{
		TS: 1,
		FixtureSummary: ScoresBatchSummary{
			FixtureID:         1,
			EventsSubTreeRoot: root,
		},
		FixtureProof:  []ProofNode{{Hash: root}},
		MainTreeProof: []ProofNode{{Hash: root}},
		Predicate:     TraderPredicate{Threshold: 1},
		StatA: StatTerm{
			StatToProve:   ScoreStat{Key: 1, Value: 1, Period: 0},
			EventStatRoot: root,
			StatProof:     []ProofNode{{Hash: root}},
		},
	}
	statB := args.StatA
	op := uint8(2)
	args.StatB = &statB
	args.Op = &op

	cases := []struct {
		name string
		fn   func(*errWriter) error
	}{
		{"validateStatArgs", func(w *errWriter) error { return writeValidateStatArgs(w, args) }},
		{"scoresBatchSummary", func(w *errWriter) error {
			return writeScoresBatchSummary(w, args.FixtureSummary)
		}},
		{"proofNodes", func(w *errWriter) error { return writeProofNodes(w, args.FixtureProof) }},
		{"proofNodesEmpty", func(w *errWriter) error { return writeProofNodes(w, nil) }},
		{"traderPredicate", func(w *errWriter) error { return writeTraderPredicate(w, args.Predicate) }},
		{"statTerm", func(w *errWriter) error { return writeStatTerm(w, args.StatA) }},
		{"optionStatTerm", func(w *errWriter) error { return writeOptionStatTerm(w, args.StatB) }},
		{"optionStatTermNil", func(w *errWriter) error { return writeOptionStatTerm(w, nil) }},
		{"optionOp", func(w *errWriter) error { return writeOptionOp(w, args.Op) }},
		{"optionOpNil", func(w *errWriter) error { return writeOptionOp(w, nil) }},
		{"settlePayload", func(w *errWriter) error {
			return encodeSettleWagerPayload(w, args, SideHome, [32]byte{1})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(&errWriter{failOn: 1}); err == nil {
				t.Fatal("expected write error")
			}
		})
	}
}

func testValidateStatArgsFixture() ValidateStatArgs {
	root := [32]byte{1}
	statB := StatTerm{
		StatToProve:   ScoreStat{Key: 2, Value: 1, Period: 0},
		EventStatRoot: root,
		StatProof:     []ProofNode{{Hash: root}},
	}
	op := uint8(1)
	return ValidateStatArgs{
		TS: 1,
		FixtureSummary: ScoresBatchSummary{
			FixtureID:         1,
			EventsSubTreeRoot: root,
			UpdateStats:       ScoresUpdateStats{UpdateCount: 1, MinTimestamp: 1, MaxTimestamp: 2},
		},
		FixtureProof:  []ProofNode{{Hash: root, IsRightSibling: true}},
		MainTreeProof: []ProofNode{{Hash: root}},
		Predicate:     TraderPredicate{Threshold: 1, Comparison: 1},
		StatA: StatTerm{
			StatToProve:   ScoreStat{Key: 1, Value: 1, Period: 0},
			EventStatRoot: root,
			StatProof:     []ProofNode{{Hash: root, IsRightSibling: true}},
		},
		StatB: &statB,
		Op:    &op,
	}
}

func TestWriteHelpersFailAtByteLimits(t *testing.T) {
	args := testValidateStatArgsFixture()
	limits := []int{0, 1, 4, 8, 12, 16, 20, 24, 32, 36, 40, 48, 52, 56, 64, 72, 96, 128, 160, 200, 256}
	cases := []struct {
		name     string
		minFails int
		fn       func(ioWriter) error
	}{
		{"validateStatArgs", 8, func(w ioWriter) error { return writeValidateStatArgs(w, args) }},
		{"statTerm", 4, func(w ioWriter) error { return writeStatTerm(w, args.StatA) }},
		{"scoresBatchSummary", 3, func(w ioWriter) error {
			return writeScoresBatchSummary(w, args.FixtureSummary)
		}},
		{"proofNodes", 2, func(w ioWriter) error { return writeProofNodes(w, args.FixtureProof) }},
		{"settlePayload", 6, func(w ioWriter) error {
			return encodeSettleWagerPayload(w, args, SideHome, [32]byte{1})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fails := 0
			for _, max := range limits {
				if err := tc.fn(&byteLimitWriter{maxBytes: max}); err != nil {
					fails++
				}
			}
			if fails < tc.minFails {
				t.Fatalf("failures = %d, want >= %d", fails, tc.minFails)
			}
		})
	}
}

type byteLimitWriter struct {
	maxBytes int
	written  int
}

func (w *byteLimitWriter) Write(p []byte) (int, error) {
	if w.written >= w.maxBytes {
		return 0, errors.New("write failed")
	}
	n := len(p)
	w.written += n
	return n, nil
}

type ioWriter interface {
	Write(p []byte) (int, error)
}

func TestEncodeSettleWagerDataPropagatesError(t *testing.T) {
	args := ValidateStatArgs{TS: 1, FixtureSummary: ScoresBatchSummary{FixtureID: 1}}
	if _, err := EncodeSettleWagerData(args, SideHome, [32]byte{}); err != nil {
		t.Fatalf("happy path: %v", err)
	}
	if err := encodeSettleWagerPayload(&errWriter{failOn: 20}, args, SideHome, [32]byte{}); err == nil {
		t.Fatal("expected payload error")
	}
}