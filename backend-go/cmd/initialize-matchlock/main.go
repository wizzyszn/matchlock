// Command initialize-matchlock creates the on-chain Config PDA on devnet.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	defaultRPC            = "https://api.devnet.solana.com"
	defaultProgramID      = "VgsUt4Fjn6jqrqP7EuqvWJM3NqYufA2haNrP9fGGaYv"
	defaultStablecoinMint = "ELWTKspHKCnCfCiCiqYw1EDH77k8VCP74dK9qytG2Ujh"
	defaultTxlineProgram  = "6pW64gN1s2uqjHkn1unFeEjAwJkPGHoppGvS715wyP2J"
)

var initializeDiscriminator = [8]byte{175, 175, 109, 31, 13, 152, 155, 237}

func main() {
	ctx := context.Background()

	rpcURL := envOr("SOLANA_RPC_URL", defaultRPC)
	programID := solana.MustPublicKeyFromBase58(envOr("MATCHLOCK_PROGRAM_ID", defaultProgramID))
	stablecoinMint := solana.MustPublicKeyFromBase58(envOr("STABLECOIN_MINT", defaultStablecoinMint))
	txlineProgram := solana.MustPublicKeyFromBase58(envOr("TXLINE_PROGRAM_ID", defaultTxlineProgram))

	keyPath := strings.TrimSpace(os.Getenv("KEEPER_KEYPAIR_PATH"))
	if keyPath == "" {
		keyPath = filepath.Join("keys", "keeper.json")
	}
	key, err := solana.PrivateKeyFromSolanaKeygenFile(keyPath)
	if err != nil {
		exitErr("load keypair: %w", err)
	}

	client := rpc.New(rpcURL)
	configPDA, _, err := solana.FindProgramAddress([][]byte{[]byte("config")}, programID)
	if err != nil {
		exitErr("derive config pda: %w", err)
	}

	if acct, err := client.GetAccountInfo(ctx, configPDA); err == nil && acct != nil && acct.Value != nil {
		fmt.Printf("Config already initialized at %s (owner=%s)\n", configPDA, acct.Value.Owner)
		return
	}

	var data []byte
	data = append(data, initializeDiscriminator[:]...)
	data = append(data, stablecoinMint[:]...)
	data = append(data, txlineProgram[:]...)

	ix := solana.NewInstruction(
		programID,
		solana.AccountMetaSlice{
			solana.Meta(key.PublicKey()).WRITE().SIGNER(),
			solana.Meta(configPDA).WRITE(),
			solana.Meta(solana.SystemProgramID),
		},
		data,
	)

	recent, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		exitErr("blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{ix},
		recent.Value.Blockhash,
		solana.TransactionPayer(key.PublicKey()),
	)
	if err != nil {
		exitErr("build tx: %w", err)
	}
	if _, err := tx.Sign(func(pk solana.PublicKey) *solana.PrivateKey {
		if pk.Equals(key.PublicKey()) {
			return &key
		}
		return nil
	}); err != nil {
		exitErr("sign tx: %w", err)
	}

	sig, err := client.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight:       false,
		PreflightCommitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		exitErr("send tx: %w", err)
	}

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		status, err := client.GetSignatureStatuses(ctx, true, sig)
		if err == nil && len(status.Value) > 0 && status.Value[0] != nil {
			if status.Value[0].Err != nil {
				exitErr("tx failed: %v", status.Value[0].Err)
			}
			if status.Value[0].ConfirmationStatus == rpc.ConfirmationStatusConfirmed ||
				status.Value[0].ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("initialize tx: %s\n", sig)
	fmt.Printf("authority: %s\n", key.PublicKey())
	fmt.Printf("config pda: %s\n", configPDA)
	fmt.Printf("stablecoin mint: %s\n", stablecoinMint)
	fmt.Printf("txline program: %s\n", txlineProgram)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitErr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
