package solana

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/matchlock/backend-go/internal/txline"
)

type ProofNode struct {
	Hash           [32]byte
	IsRightSibling bool
}

type ValidateStatArgs struct {
	TS             int64
	FixtureSummary ScoresBatchSummary
	FixtureProof   []ProofNode
	MainTreeProof  []ProofNode
	Predicate      TraderPredicate
	StatA          StatTerm
	StatB          *StatTerm
	Op             *uint8
}

type ScoresBatchSummary struct {
	FixtureID         int64
	UpdateStats       ScoresUpdateStats
	EventsSubTreeRoot [32]byte
}

type ScoresUpdateStats struct {
	UpdateCount  int32
	MinTimestamp int64
	MaxTimestamp int64
}

type ScoreStat struct {
	Key    uint32
	Value  int32
	Period int32
}

type StatTerm struct {
	StatToProve   ScoreStat
	EventStatRoot [32]byte
	StatProof     []ProofNode
}

type TraderPredicate struct {
	Threshold  int32
	Comparison uint8
}

func ValidationFromAPI(v txline.StatValidation) (ValidateStatArgs, [32]byte, error) {
	root, err := decodeHash32(string(v.Summary.EventStatsSubTreeRoot))
	if err != nil {
		return ValidateStatArgs{}, [32]byte{}, fmt.Errorf("event subtree root: %w", err)
	}
	eventRoot, err := decodeHash32(string(v.EventStatRoot))
	if err != nil {
		return ValidateStatArgs{}, [32]byte{}, fmt.Errorf("event stat root: %w", err)
	}

	args := ValidateStatArgs{
		TS: v.Summary.UpdateStats.MinTimestamp,
		FixtureSummary: ScoresBatchSummary{
			FixtureID: v.Summary.FixtureID,
			UpdateStats: ScoresUpdateStats{
				UpdateCount:  v.Summary.UpdateStats.UpdateCount,
				MinTimestamp: v.Summary.UpdateStats.MinTimestamp,
				MaxTimestamp: v.Summary.UpdateStats.MaxTimestamp,
			},
			EventsSubTreeRoot: root,
		},
		FixtureProof:  proofNodesFromAPI(v.SubTreeProof),
		MainTreeProof: proofNodesFromAPI(v.MainTreeProof),
		Predicate: TraderPredicate{
			Threshold:  0,
			Comparison: 0, // GreaterThan
		},
		StatA: StatTerm{
			StatToProve: ScoreStat{
				Key:    v.StatToProve.Key,
				Value:  v.StatToProve.Value,
				Period: v.StatToProve.Period,
			},
			EventStatRoot: eventRoot,
			StatProof:     proofNodesFromAPI(v.StatProof),
		},
	}
	return args, root, nil
}

func proofNodesFromAPI(nodes []txline.ProofNodeResponse) []ProofNode {
	out := make([]ProofNode, 0, len(nodes))
	for _, n := range nodes {
		hash, err := decodeHash32(string(n.Hash))
		if err != nil {
			continue
		}
		out = append(out, ProofNode{Hash: hash, IsRightSibling: n.IsRightSibling})
	}
	return out
}

func decodeHash32(value string) ([32]byte, error) {
	var out [32]byte
	if value == "" {
		return out, fmt.Errorf("empty hash")
	}
	if len(value) == 64 {
		raw, err := decodeHex(value)
		if err == nil && len(raw) == 32 {
			copy(out[:], raw)
			return out, nil
		}
	}
	if raw, err := base64.StdEncoding.DecodeString(value); err == nil && len(raw) == 32 {
		copy(out[:], raw)
		return out, nil
	}
	if raw, err := base64.RawStdEncoding.DecodeString(value); err == nil && len(raw) == 32 {
		copy(out[:], raw)
		return out, nil
	}
	return out, fmt.Errorf("unsupported hash encoding")
}

func decodeHex(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("odd hex length")
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		var b byte
		_, err := fmt.Sscanf(s[2*i:2*i+2], "%02x", &b)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	return out, nil
}

var settleWagerDiscriminator = [8]byte{161, 242, 169, 152, 172, 163, 161, 104}
var voidWagerDiscriminator = anchorDiscriminator("void_wager")

func EncodeSettleWagerData(validation ValidateStatArgs, winningSide uint8, merkleRoot [32]byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeResolutionPayload(&buf, settleWagerDiscriminator, validation, winningSide, merkleRoot); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeSettleWagerPayload(buf io.Writer, validation ValidateStatArgs, winningSide uint8, merkleRoot [32]byte) error {
	return encodeResolutionPayload(buf, settleWagerDiscriminator, validation, winningSide, merkleRoot)
}

func EncodeVoidWagerData(validation ValidateStatArgs, winningSide uint8, merkleRoot [32]byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeResolutionPayload(&buf, voidWagerDiscriminator, validation, winningSide, merkleRoot); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeResolutionPayload(buf io.Writer, discriminator [8]byte, validation ValidateStatArgs, winningSide uint8, merkleRoot [32]byte) error {
	if _, err := buf.Write(discriminator[:]); err != nil {
		return err
	}
	if err := writeValidateStatArgs(buf, validation); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, winningSide); err != nil {
		return err
	}
	_, err := buf.Write(merkleRoot[:])
	return err
}

func writeValidateStatArgs(buf io.Writer, v ValidateStatArgs) error {
	if err := binary.Write(buf, binary.LittleEndian, v.TS); err != nil {
		return err
	}
	if err := writeScoresBatchSummary(buf, v.FixtureSummary); err != nil {
		return err
	}
	if err := writeProofNodes(buf, v.FixtureProof); err != nil {
		return err
	}
	if err := writeProofNodes(buf, v.MainTreeProof); err != nil {
		return err
	}
	if err := writeTraderPredicate(buf, v.Predicate); err != nil {
		return err
	}
	if err := writeStatTerm(buf, v.StatA); err != nil {
		return err
	}
	if err := writeOptionStatTerm(buf, v.StatB); err != nil {
		return err
	}
	return writeOptionOp(buf, v.Op)
}

func writeScoresBatchSummary(buf io.Writer, s ScoresBatchSummary) error {
	if err := binary.Write(buf, binary.LittleEndian, s.FixtureID); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, s.UpdateStats.UpdateCount); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, s.UpdateStats.MinTimestamp); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, s.UpdateStats.MaxTimestamp); err != nil {
		return err
	}
	_, err := buf.Write(s.EventsSubTreeRoot[:])
	return err
}

func writeProofNodes(buf io.Writer, nodes []ProofNode) error {
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(nodes))); err != nil {
		return err
	}
	for _, n := range nodes {
		if _, err := buf.Write(n.Hash[:]); err != nil {
			return err
		}
		var flag uint8
		if n.IsRightSibling {
			flag = 1
		}
		if err := binary.Write(buf, binary.LittleEndian, flag); err != nil {
			return err
		}
	}
	return nil
}

func writeTraderPredicate(buf io.Writer, p TraderPredicate) error {
	if err := binary.Write(buf, binary.LittleEndian, p.Threshold); err != nil {
		return err
	}
	return binary.Write(buf, binary.LittleEndian, p.Comparison)
}

func writeStatTerm(buf io.Writer, s StatTerm) error {
	if err := binary.Write(buf, binary.LittleEndian, s.StatToProve.Key); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, s.StatToProve.Value); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, s.StatToProve.Period); err != nil {
		return err
	}
	if _, err := buf.Write(s.EventStatRoot[:]); err != nil {
		return err
	}
	return writeProofNodes(buf, s.StatProof)
}

func writeOptionStatTerm(buf io.Writer, v *StatTerm) error {
	if v == nil {
		return binary.Write(buf, binary.LittleEndian, uint8(0))
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		return err
	}
	return writeStatTerm(buf, *v)
}

func writeOptionOp(buf io.Writer, v *uint8) error {
	if v == nil {
		return binary.Write(buf, binary.LittleEndian, uint8(0))
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		return err
	}
	return binary.Write(buf, binary.LittleEndian, *v)
}
