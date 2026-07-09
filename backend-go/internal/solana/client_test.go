package solana

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gagliardetto/solana-go"
)

const (
	testProgramID = "VgsUt4Fjn6jqrqP7EuqvWJM3NqYufA2haNrP9fGGaYv"
	testMint      = "ELWTKspHKCnCfCiCiqYw1EDH77k8VCP74dK9qytG2Ujh"
	testTxlineID  = "6pW64gN1s2uqjHkn1unFeEjAwJkPGHoppGvS715wyP2J"
)

func testClient(t *testing.T, mock *mockRPC) *Client {
	t.Helper()
	client, err := NewClient(mock.URL(), testProgramID, testMint, testTxlineID)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient("http://127.0.0.1:1", "bad-program", testMint, testTxlineID); err == nil {
		t.Fatal("expected program id error")
	}
	if _, err := NewClient("http://127.0.0.1:1", testProgramID, "bad-mint", testTxlineID); err == nil {
		t.Fatal("expected mint error")
	}
	if _, err := NewClient("http://127.0.0.1:1", testProgramID, testMint, "bad-txline"); err == nil {
		t.Fatal("expected txline program error")
	}
}

func TestLoadKeeperKeypairFromFile(t *testing.T) {
	if _, err := LoadKeeperKeypairFromFile(""); err == nil {
		t.Fatal("expected empty path error")
	}
	if _, err := LoadKeeperKeypairFromFile("/no/such/file.json"); err == nil {
		t.Fatal("expected missing file error")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "keeper.json")
	wallet := solana.NewWallet()
	raw, err := json.Marshal(wallet.PrivateKey)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	got, err := LoadKeeperKeypairFromFile(path)
	if err != nil {
		t.Fatalf("LoadKeeperKeypairFromFile: %v", err)
	}
	if !got.PublicKey().Equals(wallet.PublicKey()) {
		t.Fatal("keypair mismatch")
	}
}

func TestClientPing(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	client := testClient(t, mock)

	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestGetWager(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	pubkey := solana.NewWallet().PublicKey()
	data := buildWagerAccountData(t, "17952170", WagerStatusOpen, 1_000_000)
	mock.getAccountInfo = func(key string) (json.RawMessage, error) {
		if key != pubkey.String() {
			return json.RawMessage(`{"context":{"slot":100},"value":null}`), nil
		}
		payload, _ := json.Marshal(map[string]any{
			"context": map[string]any{"slot": 100},
			"value": map[string]any{
				"data":       encodeAccountData(data),
				"executable": false,
				"lamports":   1,
				"owner":      testProgramID,
			},
		})
		return payload, nil
	}

	client := testClient(t, mock)
	wager, err := client.GetWager(context.Background(), pubkey)
	if err != nil {
		t.Fatalf("GetWager: %v", err)
	}
	if wager.MatchIDString() != "17952170" {
		t.Fatalf("match_id = %q", wager.MatchIDString())
	}

	missing, err := client.GetWager(context.Background(), solana.NewWallet().PublicKey())
	if err == nil {
		t.Fatalf("expected not found, got %#v", missing)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestListWagers(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	pubkey := solana.NewWallet().PublicKey()
	data := buildWagerAccountData(t, "17952170", WagerStatusMatched, 2_000_000)
	calls := 0
	mock.getProgramAccounts = func() json.RawMessage {
		calls++
		if calls%2 == 0 {
			return []byte(`[]`)
		}
		payload, _ := json.Marshal([]map[string]any{{
			"pubkey": pubkey.String(),
			"account": map[string]any{
				"data":       encodeAccountData(data),
				"executable": false,
				"lamports":   1,
				"owner":      testProgramID,
				"rentEpoch":  0,
			},
		}})
		return payload
	}

	client := testClient(t, mock)
	status := WagerStatusMatched
	wagers, err := client.ListWagers(context.Background(), WagerFilter{
		Status:  &status,
		MatchID: "17952170",
	})
	if err != nil {
		t.Fatalf("ListWagers: %v", err)
	}
	if len(wagers) != 1 {
		t.Fatalf("len = %d", len(wagers))
	}

	matched, err := client.ListMatchedWagers(context.Background(), "17952170")
	if err != nil || len(matched) != 1 {
		t.Fatalf("ListMatchedWagers = %#v err=%v", matched, err)
	}
}

func TestSettleWagerSuccess(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	client := testClient(t, mock)

	keeper := solana.NewWallet()
	wager := Wager{
		Pubkey:     solana.NewWallet().PublicKey(),
		Maker:      keeper.PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  SideHome,
		Status:     WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("17952170"))

	args := ValidateStatArgs{TS: 1700000000000}
	sig, err := client.SettleWager(context.Background(), SettleParams{
		Settler:     keeper.PrivateKey,
		Wager:       wager,
		Validation:  args,
		WinningSide: SideHome,
	})
	if err != nil {
		t.Fatalf("SettleWager: %v", err)
	}
	if sig.String() == "" {
		t.Fatal("expected signature")
	}
}

func TestSettleWagerIdempotentPaths(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	mock.simulateTransaction = func() (json.RawMessage, error) {
		return json.RawMessage(`{"context":{"slot":100},"value":{"err":{"InstructionError":[0,{"Custom":6001}]},"logs":["InvalidStatus"]}}`), nil
	}
	client := testClient(t, mock)
	keeper := solana.NewWallet()
	wager := Wager{
		Pubkey:     solana.NewWallet().PublicKey(),
		Maker:      keeper.PublicKey(),
		Taker:      solana.NewWallet().PublicKey(),
		MatchIDLen: 8,
		MakerSide:  SideHome,
		Status:     WagerStatusMatched,
	}
	copy(wager.MatchID[:], []byte("17952170"))

	_, err := client.SettleWager(context.Background(), SettleParams{
		Settler:     keeper.PrivateKey,
		Wager:       wager,
		Validation:  ValidateStatArgs{TS: 1700000000000},
		WinningSide: SideHome,
	})
	if !errors.Is(err, ErrAlreadySettled) {
		t.Fatalf("err = %v, want ErrAlreadySettled", err)
	}

	mock.simulateTransaction = func() (json.RawMessage, error) {
		return json.RawMessage(`{"context":{"slot":100},"value":{"err":null,"logs":[]}}`), nil
	}
	mock.sendTransaction = func() (json.RawMessage, error) {
		return nil, errors.New("InvalidStatus already in use")
	}
	_, err = client.SettleWager(context.Background(), SettleParams{
		Settler:     keeper.PrivateKey,
		Wager:       wager,
		Validation:  ValidateStatArgs{TS: 1700000000000},
		WinningSide: SideHome,
	})
	if !errors.Is(err, ErrAlreadySettled) {
		t.Fatalf("send err = %v, want ErrAlreadySettled", err)
	}
}

func TestSettleWagerWinnerError(t *testing.T) {
	mock := newMockRPC()
	defer mock.Close()
	client := testClient(t, mock)
	wager := Wager{Pubkey: solana.NewWallet().PublicKey(), MakerSide: SideHome}
	_, err := client.SettleWager(context.Background(), SettleParams{
		Settler:     solana.NewWallet().PrivateKey,
		Wager:       wager,
		Validation:  ValidateStatArgs{TS: 1},
		WinningSide: 9,
	})
	if err == nil {
		t.Fatal("expected winner error")
	}
}
