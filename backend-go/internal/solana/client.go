package solana

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
)

// Client wraps Solana RPC operations for the keeper.
type Client struct {
	rpc        *rpc.Client
	programID  solana.PublicKey
	mint       solana.PublicKey
	txlineProg solana.PublicKey
}

// WagerFilter narrows program-account scans for wager listing.
type WagerFilter struct {
	Status  *uint8
	MatchID string
	Wallet  string // base58; return wagers where maker or taker matches
}

// TxlineProgramID returns the configured TxLINE program pubkey.
func (c *Client) TxlineProgramID() solana.PublicKey {
	return c.txlineProg
}

// ProgramID returns the Matchlock program pubkey.
func (c *Client) ProgramID() solana.PublicKey {
	return c.programID
}

// StablecoinMint returns the configured stablecoin mint.
func (c *Client) StablecoinMint() solana.PublicKey {
	return c.mint
}

func NewClient(rpcURL, programID, mint, txlineProgram string) (*Client, error) {
	prog, err := solana.PublicKeyFromBase58(programID)
	if err != nil {
		return nil, fmt.Errorf("program id: %w", err)
	}
	mintPK, err := solana.PublicKeyFromBase58(mint)
	if err != nil {
		return nil, fmt.Errorf("mint: %w", err)
	}
	txlinePK, err := solana.PublicKeyFromBase58(txlineProgram)
	if err != nil {
		return nil, fmt.Errorf("txline program: %w", err)
	}
	return &Client{
		rpc:        rpc.New(rpcURL),
		programID:  prog,
		mint:       mintPK,
		txlineProg: txlinePK,
	}, nil
}

func LoadKeeperKeypairFromFile(path string) (solana.PrivateKey, error) {
	if path == "" {
		return nil, fmt.Errorf("keeper keypair path is empty")
	}
	if key, err := solana.PrivateKeyFromSolanaKeygenFile(path); err == nil {
		return key, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read keeper keypair: %w", err)
	}
	return solana.PrivateKeyFromSolanaKeygenFileBytes(raw)
}

// Ping verifies the Solana RPC endpoint is reachable.
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := c.rpc.GetSlot(ctx, rpc.CommitmentProcessed)
	if err != nil {
		return fmt.Errorf("rpc get slot: %w", err)
	}
	return nil
}

// GetWager fetches and decodes a single wager account.
func (c *Client) GetWager(ctx context.Context, pubkey solana.PublicKey) (Wager, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	acct, err := c.rpc.GetAccountInfo(ctx, pubkey)
	if err != nil {
		return Wager{}, fmt.Errorf("get account %s: %w", pubkey, err)
	}
	if acct == nil || acct.Value == nil {
		return Wager{}, fmt.Errorf("wager account %s not found", pubkey)
	}
	return DecodeWager(pubkey, acct.Value.Data.GetBinary())
}

// ListWagers returns wagers indexed from chain with optional filters.
func (c *Client) ListWagers(ctx context.Context, filter WagerFilter) ([]Wager, error) {
	layouts := []struct {
		name       string
		size       uint64
		statusOff  uint64
		matchIDOff uint64
	}{
		{name: "v1", size: wagerAccountSizeV1, statusOff: statusOffsetV1, matchIDOff: matchIDOffsetV1},
		{name: "v2", size: wagerAccountSizeV2, statusOff: statusOffsetV2, matchIDOff: matchIDOffsetV2},
		{name: "v3", size: wagerAccountSizeV3, statusOff: statusOffsetV3, matchIDOff: matchIDOffsetV3},
		{name: "v4", size: wagerAccountSizeV4, statusOff: statusOffsetV4, matchIDOff: matchIDOffsetV4},
	}

	accountsByLayout := make([][]*rpc.KeyedAccount, len(layouts))
	errs := make([]error, len(layouts))
	var wg sync.WaitGroup
	for i, layout := range layouts {
		wg.Add(1)
		go func(i int, layout struct {
			name       string
			size       uint64
			statusOff  uint64
			matchIDOff uint64
		}) {
			defer wg.Done()
			filters := wagerAccountFilters(layout.size, layout.statusOff, layout.matchIDOff, filter)
			accountsByLayout[i], errs[i] = c.rpc.GetProgramAccountsWithOpts(ctx, c.programID, &rpc.GetProgramAccountsOpts{
				Filters: filters,
			})
		}(i, layout)
	}
	wg.Wait()

	var accounts []*rpc.KeyedAccount
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("get %s program accounts: %w", layouts[i].name, err)
		}
		accounts = append(accounts, accountsByLayout[i]...)
	}
	out := make([]Wager, 0, len(accounts))
	for _, acct := range accounts {
		w, err := DecodeWager(acct.Pubkey, acct.Account.Data.GetBinary())
		if err != nil {
			continue
		}
		if filter.MatchID != "" && w.MatchIDString() != filter.MatchID {
			continue
		}
		if filter.Status != nil && w.Status != *filter.Status {
			continue
		}
		if filter.Wallet != "" {
			walletPK, err := solana.PublicKeyFromBase58(filter.Wallet)
			if err != nil {
				continue
			}
			if !w.Maker.Equals(walletPK) && !w.Taker.Equals(walletPK) {
				continue
			}
		}
		out = append(out, w)
	}
	return out, nil
}

func wagerAccountFilters(size, statusOffset, matchIDOffset uint64, filter WagerFilter) []rpc.RPCFilter {
	filters := []rpc.RPCFilter{
		{DataSize: size},
		{Memcmp: &rpc.RPCFilterMemcmp{Offset: 0, Bytes: wagerDiscriminator[:]}},
	}
	if filter.Status != nil {
		filters = append(filters, rpc.RPCFilter{
			Memcmp: &rpc.RPCFilterMemcmp{Offset: statusOffset, Bytes: []byte{*filter.Status}},
		})
	}
	if filter.MatchID != "" {
		filters = append(filters, rpc.RPCFilter{
			Memcmp: &rpc.RPCFilterMemcmp{Offset: matchIDOffset, Bytes: MatchIDFilterBytes(filter.MatchID)},
		})
	}
	return filters
}

// ListMatchedWagers returns matched wagers for a given match_id string.
func (c *Client) ListMatchedWagers(ctx context.Context, matchID string) ([]Wager, error) {
	status := WagerStatusMatched
	return c.ListWagers(ctx, WagerFilter{Status: &status, MatchID: matchID})
}

type SettleParams struct {
	Settler     solana.PrivateKey
	Wager       Wager
	Validation  ValidateStatArgs
	MerkleRoot  [32]byte
	WinningSide uint8
}

func (c *Client) SettleWager(ctx context.Context, p SettleParams) (solana.Signature, error) {
	winner, err := p.Wager.WinnerPubkey(p.WinningSide)
	if err != nil {
		return solana.Signature{}, err
	}

	configPDA, _, err := FindConfigPDA(c.programID)
	if err != nil {
		return solana.Signature{}, err
	}
	vaultPDA, _, err := FindVaultPDA(c.programID, p.Wager.Pubkey)
	if err != nil {
		return solana.Signature{}, err
	}
	epochDay := EpochDayFromMillis(p.Validation.TS)
	dailyScores, _, err := FindDailyScoresRootsPDA(c.txlineProg, epochDay)
	if err != nil {
		return solana.Signature{}, err
	}

	winnerATA, _, err := solana.FindAssociatedTokenAddress(winner, c.mint)
	if err != nil {
		return solana.Signature{}, err
	}

	ixData, err := EncodeSettleWagerData(p.Validation, p.WinningSide, p.MerkleRoot)
	if err != nil {
		return solana.Signature{}, err
	}

	accounts := solana.AccountMetaSlice{
		solana.Meta(p.Settler.PublicKey()).SIGNER().WRITE(),
		solana.Meta(configPDA),
		solana.Meta(p.Wager.Pubkey).WRITE(),
		solana.Meta(vaultPDA).WRITE(),
		solana.Meta(winner).WRITE(),
		solana.Meta(winnerATA).WRITE(),
		solana.Meta(c.mint),
		solana.Meta(dailyScores),
		solana.Meta(c.txlineProg),
		solana.Meta(token.ProgramID),
		solana.Meta(associatedtokenaccount.ProgramID),
	}

	ix := solana.NewInstruction(c.programID, accounts, ixData)
	latest, err := c.rpc.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("latest blockhash: %w", err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{setComputeUnitLimit(1_400_000), ix},
		latest.Value.Blockhash,
		solana.TransactionPayer(p.Settler.PublicKey()),
	)
	if err != nil {
		return solana.Signature{}, err
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(p.Settler.PublicKey()) {
			k := p.Settler
			return &k
		}
		return nil
	})
	if err != nil {
		return solana.Signature{}, fmt.Errorf("sign tx: %w", err)
	}

	sim, err := c.rpc.SimulateTransaction(ctx, tx)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("simulate: %w", err)
	}
	if sim.Value.Err != nil {
		if isIdempotentSettleError(sim.Value.Err) {
			return solana.Signature{}, ErrAlreadySettled
		}
		return solana.Signature{}, fmt.Errorf("simulation failed: %v logs=%v", sim.Value.Err, sim.Value.Logs)
	}

	sig, err := c.rpc.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight:       true,
		PreflightCommitment: rpc.CommitmentProcessed,
	})
	if err != nil {
		if isIdempotentSendError(err) {
			return solana.Signature{}, ErrAlreadySettled
		}
		return solana.Signature{}, fmt.Errorf("send tx: %w", err)
	}

	confirmCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if err := waitForSignature(confirmCtx, c.rpc, sig); err != nil {
		return sig, fmt.Errorf("confirm tx %s: %w", sig, err)
	}
	return sig, nil
}

func waitForSignature(ctx context.Context, client *rpc.Client, sig solana.Signature) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		statuses, err := client.GetSignatureStatuses(ctx, true, sig)
		if err == nil && statuses != nil && len(statuses.Value) > 0 && statuses.Value[0] != nil {
			st := statuses.Value[0]
			if st.Err != nil {
				return fmt.Errorf("transaction failed: %v", st.Err)
			}
			if st.ConfirmationStatus == rpc.ConfirmationStatusConfirmed ||
				st.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

var ErrAlreadySettled = fmt.Errorf("wager already settled")

func isIdempotentSettleError(err interface{}) bool {
	s := fmt.Sprint(err)
	return containsAny(s, "InvalidStatus", "6001", "already settled", "AccountNotFound")
}

func isIdempotentSendError(err error) bool {
	if err == nil {
		return false
	}
	return containsAny(err.Error(), "InvalidStatus", "6001", "already in use")
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if len(p) > 0 && stringContains(s, p) {
			return true
		}
	}
	return false
}

func stringContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
