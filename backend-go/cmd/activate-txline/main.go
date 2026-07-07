// Command activate-txline subscribes to the TxLINE World Cup free tier on devnet
// and activates an API token for backend-go.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	devnetAPIOrigin   = "https://txline-dev.txodds.com"
	devnetRPC         = "https://api.devnet.solana.com"
	txlineProgramID   = "6pW64gN1s2uqjHkn1unFeEjAwJkPGHoppGvS715wyP2J"
	txlTokenMint      = "4Zao8ocPhmMgq7PdsYWyxvqySMGx7xb9cMftPMkEokRG"
	serviceLevelID    = uint16(1) // World Cup + Int Friendlies (60s delay)
	durationWeeks     = uint8(4)
	minBalanceLamports = 10_000_000 // 0.01 SOL
)

var subscribeDiscriminator = [8]byte{254, 28, 191, 138, 156, 179, 183, 53}

func main() {
	ctx := context.Background()

	keyPath := strings.TrimSpace(os.Getenv("KEEPER_KEYPAIR_PATH"))
	if keyPath == "" {
		keyPath = filepath.Join("keys", "keeper.json")
	}
	key, err := loadKeypair(keyPath)
	if err != nil {
		exitErr("load keypair: %w", err)
	}

	client := rpc.New(devnetRPC)
	if err := ensureBalance(ctx, client, key.PublicKey()); err != nil {
		exitErr("fund wallet: %w", err)
	}

	jwt, err := guestJWT(ctx)
	if err != nil {
		exitErr("guest jwt: %w", err)
	}

	txSig, err := subscribeFreeTier(ctx, client, key)
	if err != nil {
		exitErr("subscribe: %w", err)
	}
	fmt.Fprintf(os.Stderr, "subscribe tx: %s\n", txSig)

	apiToken, err := activateToken(ctx, jwt, txSig, key)
	if err != nil {
		exitErr("activate: %w", err)
	}

	if err := writeEnvToken(apiToken); err != nil {
		exitErr("write .env: %w", err)
	}

	fmt.Printf("TxLINE API token activated for %s\n", key.PublicKey())
	fmt.Printf("Wallet: %s\n", key.PublicKey())
	fmt.Println("Updated backend-go/.env with TXLINE_API_TOKEN")
}

func loadKeypair(path string) (solana.PrivateKey, error) {
	if _, err := os.Stat(path); err != nil {
		home, _ := os.UserHomeDir()
		fallback := filepath.Join(home, ".config", "solana", "id.json")
		if path != fallback {
			if key, err2 := solana.PrivateKeyFromSolanaKeygenFile(fallback); err2 == nil {
				return key, nil
			}
		}
		return nil, fmt.Errorf("keypair %s: %w", path, err)
	}
	return solana.PrivateKeyFromSolanaKeygenFile(path)
}

func ensureBalance(ctx context.Context, client *rpc.Client, wallet solana.PublicKey) error {
	bal, err := client.GetBalance(ctx, wallet, rpc.CommitmentConfirmed)
	if err != nil {
		return err
	}
	if bal.Value >= minBalanceLamports {
		return nil
	}

	fmt.Fprintf(os.Stderr, "balance %.4f SOL — requesting devnet airdrop...\n", float64(bal.Value)/1e9)
	for attempt := 1; attempt <= 6; attempt++ {
		sig, err := client.RequestAirdrop(ctx, wallet, 2*solana.LAMPORTS_PER_SOL, rpc.CommitmentConfirmed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "airdrop attempt %d: %v\n", attempt, err)
		} else {
			fmt.Fprintf(os.Stderr, "airdrop sig: %s\n", sig)
		}
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			bal, err = client.GetBalance(ctx, wallet, rpc.CommitmentConfirmed)
			if err == nil && bal.Value >= minBalanceLamports {
				return nil
			}
			time.Sleep(2 * time.Second)
		}
		time.Sleep(time.Duration(attempt) * 5 * time.Second)
	}
	return fmt.Errorf("insufficient devnet SOL on %s; fund via https://faucet.solana.com then re-run", wallet)
}

func guestJWT(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, devnetAPIOrigin+"/auth/guest/start", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("guest auth status=%d body=%s", resp.StatusCode, string(body))
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", fmt.Errorf("guest auth missing token")
	}
	return out.Token, nil
}

func subscribeFreeTier(ctx context.Context, client *rpc.Client, payer solana.PrivateKey) (string, error) {
	programID := solana.MustPublicKeyFromBase58(txlineProgramID)
	mint := solana.MustPublicKeyFromBase58(txlTokenMint)

	pricingMatrix, _, err := solana.FindProgramAddress([][]byte{[]byte("pricing_matrix")}, programID)
	if err != nil {
		return "", err
	}
	tokenTreasuryPDA, _, err := solana.FindProgramAddress([][]byte{[]byte("token_treasury_v2")}, programID)
	if err != nil {
		return "", err
	}
	tokenTreasuryVault, _, err := solana.FindAssociatedTokenAddressWithProgram(
		tokenTreasuryPDA, mint, solana.Token2022ProgramID,
	)
	if err != nil {
		return "", err
	}
	userATA, _, err := solana.FindAssociatedTokenAddressWithProgram(
		payer.PublicKey(), mint, solana.Token2022ProgramID,
	)
	if err != nil {
		return "", err
	}

	var data []byte
	data = append(data, subscribeDiscriminator[:]...)
	var buf [3]byte
	binary.LittleEndian.PutUint16(buf[0:2], serviceLevelID)
	buf[2] = durationWeeks
	data = append(data, buf[:]...)

	accounts := solana.AccountMetaSlice{
		solana.Meta(payer.PublicKey()).SIGNER().WRITE(),
		solana.Meta(pricingMatrix),
		solana.Meta(mint),
		solana.Meta(userATA).WRITE(),
		solana.Meta(tokenTreasuryVault).WRITE(),
		solana.Meta(tokenTreasuryPDA),
		solana.Meta(solana.Token2022ProgramID),
		solana.Meta(solana.SystemProgramID),
		solana.Meta(associatedtokenaccount.ProgramID),
	}

	createATA := associatedtokenaccount.NewCreateInstructionWithTokenProgram(
		payer.PublicKey(),
		payer.PublicKey(),
		mint,
		solana.Token2022ProgramID,
	).Build()

	subscribeIX := solana.NewInstruction(programID, accounts, data)

	latest, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return "", err
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{createATA, subscribeIX},
		latest.Value.Blockhash,
		solana.TransactionPayer(payer.PublicKey()),
	)
	if err != nil {
		return "", err
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payer.PublicKey()) {
			k := payer
			return &k
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	sig, err := client.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight:       false,
		PreflightCommitment: rpc.CommitmentProcessed,
	})
	if err != nil {
		return "", err
	}

	confirmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	for {
		statuses, err := client.GetSignatureStatuses(confirmCtx, true, sig)
		if err == nil && statuses != nil && len(statuses.Value) > 0 && statuses.Value[0] != nil {
			if statuses.Value[0].Err != nil {
				return "", fmt.Errorf("subscribe failed: %v", statuses.Value[0].Err)
			}
			if statuses.Value[0].ConfirmationStatus == rpc.ConfirmationStatusConfirmed ||
				statuses.Value[0].ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				return sig.String(), nil
			}
		}
		select {
		case <-confirmCtx.Done():
			return sig.String(), confirmCtx.Err()
		case <-time.After(time.Second):
		}
	}
}

func activateToken(ctx context.Context, jwt, txSig string, key solana.PrivateKey) (string, error) {
	message := []byte(txSig + "::" + jwt)
	sig, err := key.Sign(message)
	if err != nil {
		return "", err
	}
	payload, _ := json.Marshal(map[string]any{
		"txSig":           txSig,
		"walletSignature": base64.StdEncoding.EncodeToString(sig[:]),
		"leagues":         []int{},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, devnetAPIOrigin+"/api/token/activate", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("activate status=%d body=%s", resp.StatusCode, string(body))
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return strings.TrimSpace(string(body)), nil
	}
	if raw, ok := obj["token"]; ok {
		var token string
		if err := json.Unmarshal(raw, &token); err == nil && token != "" {
			return token, nil
		}
	}
	return strings.TrimSpace(string(body)), nil
}

func writeEnvToken(token string) error {
	envPath := ".env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := os.Chdir("backend-go"); err == nil {
			envPath = ".env"
		}
	}
	raw, err := os.ReadFile(envPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	updated := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "TXLINE_API_TOKEN") {
			lines[i] = "TXLINE_API_TOKEN=" + token
			updated = true
			break
		}
	}
	if !updated {
		lines = append(lines, "TXLINE_API_TOKEN="+token)
	}
	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func exitErr(format string, err error) {
	fmt.Fprintf(os.Stderr, "activate-txline: "+format+"\n", err)
	os.Exit(1)
}