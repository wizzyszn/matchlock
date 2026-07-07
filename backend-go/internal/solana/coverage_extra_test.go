package solana

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/matchlock/backend-go/internal/txline"
)

func TestListWagersSkipsInvalidAccounts(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	validPubkey := solana.NewWallet().PublicKey()
	validData := buildWagerAccountData(t, "17952170", WagerStatusMatched, 1)
	calls := 0
	mock.getProgramAccounts = func() json.RawMessage {
		calls++
		if calls > 1 {
			return []byte(`[]`)
		}
		payload, _ := json.Marshal([]map[string]any{
			{
				"pubkey": solana.NewWallet().PublicKey().String(),
				"account": map[string]any{
					"data": encodeAccountData([]byte{1, 2, 3}),
					"executable": false, "lamports": 1, "owner": testProgramID, "rentEpoch": 0,
				},
			},
			{
				"pubkey": validPubkey.String(),
				"account": map[string]any{
					"data": encodeAccountData(validData),
					"executable": false, "lamports": 1, "owner": testProgramID, "rentEpoch": 0,
				},
			},
		})
		return payload
	}
	client := testClient(t, mock)
	wagers, err := client.ListWagers(context.Background(), WagerFilter{MatchID: "17952170"})
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(wagers) != 1 {
		t.Fatalf("len = %d, want 1", len(wagers))
	}
}

func TestWagerWinnerPubkeySides(t *testing.T) {
	maker := solana.NewWallet().PublicKey()
	taker := solana.NewWallet().PublicKey()
	wager := Wager{Maker: maker, Taker: taker, MakerSide: SideHome, TakerSide: SideAway}

	winner, err := wager.WinnerPubkey(SideHome)
	if err != nil || !winner.Equals(maker) {
		t.Fatalf("maker winner = %v err=%v", winner, err)
	}
	winner, err = wager.WinnerPubkey(SideAway)
	if err != nil || !winner.Equals(taker) {
		t.Fatalf("taker winner = %v err=%v", winner, err)
	}
}

func TestLoadKeeperKeypairFromFileInvalidBytes(t *testing.T) {
	path := t.TempDir() + "/bad.json"
	if err := os.WriteFile(path, []byte(`[1,2,3]`), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := LoadKeeperKeypairFromFile(path); err == nil {
		t.Fatal("expected invalid keypair bytes error")
	}
}

func TestWaitForSignatureFinalized(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.signatureStatuses = func() json.RawMessage {
		return json.RawMessage(`{"context":{"slot":100},"value":[{"slot":100,"err":null,"confirmationStatus":"finalized"}]}`)
	}
	client := testClient(t, mock)
	var sig solana.Signature
	sig[0] = 7
	if err := waitForSignature(context.Background(), client.rpc, sig); err != nil {
		t.Fatalf("waitForSignature: %v", err)
	}
}

func TestEncodeSettleWagerDataMinimal(t *testing.T) {
	args := ValidateStatArgs{
		TS: 1,
		FixtureSummary: ScoresBatchSummary{FixtureID: 1},
		StatA:          StatTerm{StatToProve: ScoreStat{Key: 1}},
	}
	if _, err := EncodeSettleWagerData(args, SideAway, [32]byte{4}); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func TestListWagersNoFilters(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	pubkey := solana.NewWallet().PublicKey()
	data := buildWagerAccountData(t, "17952170", WagerStatusOpen, 1)
	calls := 0
	mock.getProgramAccounts = func() json.RawMessage {
		calls++
		if calls > 1 {
			return []byte(`[]`)
		}
		payload, _ := json.Marshal([]map[string]any{{
			"pubkey": pubkey.String(),
			"account": map[string]any{
				"data": encodeAccountData(data), "executable": false,
				"lamports": 1, "owner": testProgramID, "rentEpoch": 0,
			},
		}})
		return payload
	}
	client := testClient(t, mock)
	wagers, err := client.ListWagers(context.Background(), WagerFilter{})
	if err != nil || len(wagers) != 1 {
		t.Fatalf("wagers=%#v err=%v", wagers, err)
	}
}

func TestGetWagerNilAccountValue(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.getAccountInfo = func(string) (json.RawMessage, error) {
		return json.RawMessage(`{"context":{"slot":100},"value":null}`), nil
	}
	client := testClient(t, mock)
	if _, err := client.GetWager(context.Background(), solana.NewWallet().PublicKey()); err == nil {
		t.Fatal("expected not found")
	}
}

func TestListWagersStatusFilterMismatch(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	pubkey := solana.NewWallet().PublicKey()
	data := buildWagerAccountData(t, "17952170", WagerStatusOpen, 1)
	mock.getProgramAccounts = func() json.RawMessage {
		payload, _ := json.Marshal([]map[string]any{{
			"pubkey": pubkey.String(),
			"account": map[string]any{
				"data": encodeAccountData(data), "executable": false,
				"lamports": 1, "owner": testProgramID, "rentEpoch": 0,
			},
		}})
		return payload
	}
	client := testClient(t, mock)
	status := WagerStatusMatched
	wagers, err := client.ListWagers(context.Background(), WagerFilter{Status: &status})
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(wagers) != 0 {
		t.Fatalf("len = %d", len(wagers))
	}
}

func TestSettleWagerAwayWinnerSuccess(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	client := testClient(t, mock)
	keeper := solana.NewWallet()
	wager := matchedWager(keeper.PublicKey())
	wager.MakerSide = SideHome

	sig, err := client.SettleWager(context.Background(), SettleParams{
		Settler: keeper.PrivateKey, Wager: wager,
		Validation: ValidateStatArgs{TS: 1700000000000}, WinningSide: SideAway,
	})
	if err != nil {
		t.Fatalf("SettleWager: %v", err)
	}
	if sig.IsZero() {
		t.Fatal("expected signature")
	}
}

func TestEncodeSettleWagerDataFromFixture(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "txline_proof_response.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var validation txline.StatValidation
	if err := json.Unmarshal(raw, &validation); err != nil {
		t.Fatalf("decode: %v", err)
	}
	args, root, err := ValidationFromAPI(validation)
	if err != nil {
		t.Fatalf("ValidationFromAPI: %v", err)
	}
	data, err := EncodeSettleWagerData(args, SideHome, root)
	if err != nil || len(data) < 64 {
		t.Fatalf("encode len=%d err=%v", len(data), err)
	}
}

func TestSettleWagerConfirmTxFailure(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.signatureStatuses = func() json.RawMessage {
		return json.RawMessage(`{"context":{"slot":100},"value":[{"slot":100,"err":{"InstructionError":[0,{"Custom":1}]},"confirmationStatus":"confirmed"}]}`)
	}
	client := testClient(t, mock)
	keeper := solana.NewWallet()
	_, err := client.SettleWager(context.Background(), SettleParams{
		Settler: keeper.PrivateKey, Wager: matchedWager(keeper.PublicKey()),
		Validation: ValidateStatArgs{TS: 1700000000000}, WinningSide: SideHome,
	})
	if err == nil {
		t.Fatal("expected confirm failure")
	}
}

func TestSettleWagerCanceledDuringConfirm(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.signatureStatuses = func() json.RawMessage {
		return json.RawMessage(`{"context":{"slot":100},"value":[null]}`)
	}
	client := testClient(t, mock)
	keeper := solana.NewWallet()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.SettleWager(ctx, SettleParams{
		Settler: keeper.PrivateKey, Wager: matchedWager(keeper.PublicKey()),
		Validation: ValidateStatArgs{TS: 1700000000000}, WinningSide: SideHome,
	})
	if err == nil {
		t.Fatal("expected context error")
	}
}