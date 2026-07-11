package solana

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
)

var (
	makeWagerDiscriminator      = [8]byte{14, 41, 8, 64, 67, 76, 114, 149}
	acceptWagerDiscriminator    = [8]byte{214, 18, 178, 214, 203, 22, 50, 119}
	cancelWagerDiscriminator    = [8]byte{57, 92, 124, 123, 216, 16, 37, 148}
	faucetDiscriminator         = [8]byte{49, 178, 104, 8, 23, 120, 186, 21}
	registerWalletDiscriminator = [8]byte{181, 141, 102, 82, 135, 213, 141, 8}
)

func FindWagerPDA(programID, maker solana.PublicKey, matchID []byte, nonce uint64) (solana.PublicKey, uint8, error) {
	nonceLE := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceLE, nonce)
	return solana.FindProgramAddress([][]byte{[]byte(WagerSeed), maker.Bytes(), matchID, nonceLE}, programID)
}

func EncodeMakeWagerData(matchID []byte, stake uint64, makerSide uint8, invitedTaker solana.PublicKey, participant1IsHome bool, nonce uint64) ([]byte, error) {
	if len(matchID) == 0 || len(matchID) > 32 {
		return nil, fmt.Errorf("match_id length %d out of range", len(matchID))
	}
	if _, err := strconv.ParseInt(string(matchID), 10, 64); err != nil {
		return nil, fmt.Errorf("match_id must be a numeric fixture id")
	}
	var buf []byte
	buf = append(buf, makeWagerDiscriminator[:]...)
	var lenLE [4]byte
	binary.LittleEndian.PutUint32(lenLE[:], uint32(len(matchID)))
	buf = append(buf, lenLE[:]...)
	buf = append(buf, matchID...)
	var stakeLE [8]byte
	binary.LittleEndian.PutUint64(stakeLE[:], stake)
	buf = append(buf, stakeLE[:]...)
	buf = append(buf, makerSide)
	buf = append(buf, invitedTaker.Bytes()...)
	if participant1IsHome {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	var nonceLE [8]byte
	binary.LittleEndian.PutUint64(nonceLE[:], nonce)
	buf = append(buf, nonceLE[:]...)
	return buf, nil
}

func EncodeAcceptWagerData(takerSide uint8) []byte {
	buf := make([]byte, len(acceptWagerDiscriminator)+1)
	copy(buf, acceptWagerDiscriminator[:])
	buf[len(acceptWagerDiscriminator)] = takerSide
	return buf
}

func EncodeCancelWagerData() []byte {
	return cancelWagerDiscriminator[:]
}

type MakeWagerParams struct {
	Maker              solana.PrivateKey
	MatchID            string
	Stake              uint64
	MakerSide          uint8
	InvitedTaker       solana.PublicKey
	Participant1IsHome bool
	Nonce              uint64
}

func (c *Client) MakeWager(ctx context.Context, p MakeWagerParams) (solana.PublicKey, solana.Signature, error) {
	matchBytes := []byte(p.MatchID)
	configPDA, _, err := FindConfigPDA(c.programID)
	if err != nil {
		return solana.PublicKey{}, solana.Signature{}, err
	}
	wagerPDA, _, err := FindWagerPDA(c.programID, p.Maker.PublicKey(), matchBytes, p.Nonce)
	if err != nil {
		return solana.PublicKey{}, solana.Signature{}, err
	}
	vaultPDA, _, err := FindVaultPDA(c.programID, wagerPDA)
	if err != nil {
		return solana.PublicKey{}, solana.Signature{}, err
	}
	makerATA, _, err := solana.FindAssociatedTokenAddress(p.Maker.PublicKey(), c.mint)
	if err != nil {
		return solana.PublicKey{}, solana.Signature{}, err
	}

	ixData, err := EncodeMakeWagerData(matchBytes, p.Stake, p.MakerSide, p.InvitedTaker, p.Participant1IsHome, p.Nonce)
	if err != nil {
		return solana.PublicKey{}, solana.Signature{}, err
	}

	accounts := solana.AccountMetaSlice{
		solana.Meta(p.Maker.PublicKey()).SIGNER().WRITE(),
		solana.Meta(configPDA),
		solana.Meta(wagerPDA).WRITE(),
		solana.Meta(vaultPDA).WRITE(),
		solana.Meta(makerATA).WRITE(),
		solana.Meta(c.mint),
		solana.Meta(token.ProgramID),
		solana.Meta(associatedtokenaccount.ProgramID),
		solana.Meta(solana.SystemProgramID),
	}
	ix := solana.NewInstruction(c.programID, accounts, ixData)
	_, sig, err := c.sendSigned(ctx, p.Maker, []solana.Instruction{ix})
	return wagerPDA, sig, err
}

type CancelWagerParams struct {
	Maker solana.PrivateKey
	Wager solana.PublicKey
}

func (c *Client) CancelWager(ctx context.Context, p CancelWagerParams) (solana.Signature, error) {
	configPDA, _, err := FindConfigPDA(c.programID)
	if err != nil {
		return solana.Signature{}, err
	}
	vaultPDA, _, err := FindVaultPDA(c.programID, p.Wager)
	if err != nil {
		return solana.Signature{}, err
	}
	makerATA, _, err := solana.FindAssociatedTokenAddress(p.Maker.PublicKey(), c.mint)
	if err != nil {
		return solana.Signature{}, err
	}

	accounts := solana.AccountMetaSlice{
		solana.Meta(p.Maker.PublicKey()).SIGNER().WRITE(),
		solana.Meta(configPDA),
		solana.Meta(p.Wager).WRITE(),
		solana.Meta(vaultPDA).WRITE(),
		solana.Meta(makerATA).WRITE(),
		solana.Meta(c.mint),
		solana.Meta(token.ProgramID),
		solana.Meta(associatedtokenaccount.ProgramID),
	}
	ix := solana.NewInstruction(c.programID, accounts, EncodeCancelWagerData())
	_, sig, err := c.sendSigned(ctx, p.Maker, []solana.Instruction{ix})
	return sig, err
}

type AcceptWagerParams struct {
	Taker     solana.PrivateKey
	Wager     solana.PublicKey
	Maker     solana.PublicKey
	TakerSide uint8
}

func (c *Client) AcceptWager(ctx context.Context, p AcceptWagerParams) (solana.Signature, error) {
	configPDA, _, err := FindConfigPDA(c.programID)
	if err != nil {
		return solana.Signature{}, err
	}
	vaultPDA, _, err := FindVaultPDA(c.programID, p.Wager)
	if err != nil {
		return solana.Signature{}, err
	}
	takerATA, _, err := solana.FindAssociatedTokenAddress(p.Taker.PublicKey(), c.mint)
	if err != nil {
		return solana.Signature{}, err
	}

	accounts := solana.AccountMetaSlice{
		solana.Meta(p.Taker.PublicKey()).SIGNER().WRITE(),
		solana.Meta(configPDA),
		solana.Meta(p.Wager).WRITE(),
		solana.Meta(p.Maker),
		solana.Meta(takerATA).WRITE(),
		solana.Meta(vaultPDA).WRITE(),
		solana.Meta(c.mint),
		solana.Meta(token.ProgramID),
		solana.Meta(associatedtokenaccount.ProgramID),
	}
	ix := solana.NewInstruction(c.programID, accounts, EncodeAcceptWagerData(p.TakerSide))
	_, sig, err := c.sendSigned(ctx, p.Taker, []solana.Instruction{ix})
	return sig, err
}

// EncodeRegisterWalletData builds instruction data for the register_wallet ix.
func EncodeRegisterWalletData(wallet solana.PublicKey, userIDHash [32]byte) []byte {
	buf := make([]byte, 0, 8+32+32)
	buf = append(buf, registerWalletDiscriminator[:]...)
	buf = append(buf, wallet.Bytes()...)
	buf = append(buf, userIDHash[:]...)
	return buf
}

// RegisterWallet creates the on-chain WalletProfile PDA for a user's wallet.
// The keeper authority must sign because register_wallet requires config.authority.
func (c *Client) RegisterWallet(ctx context.Context, keeperKey solana.PrivateKey, wallet solana.PublicKey, userIDHash [32]byte) error {
	configPDA, _, err := FindConfigPDA(c.programID)
	if err != nil {
		return fmt.Errorf("config pda: %w", err)
	}
	walletProfilePDA, _, err := FindWalletProfilePDA(c.programID, wallet)
	if err != nil {
		return fmt.Errorf("wallet profile pda: %w", err)
	}

	ixData := EncodeRegisterWalletData(wallet, userIDHash)
	accounts := solana.AccountMetaSlice{
		solana.Meta(keeperKey.PublicKey()).SIGNER().WRITE(), // authority
		solana.Meta(configPDA),                              // config
		solana.Meta(walletProfilePDA).WRITE(),               // wallet_profile (init)
		solana.Meta(solana.SystemProgramID),                 // system_program
	}
	ix := solana.NewInstruction(c.programID, accounts, ixData)
	_, _, err = c.sendSigned(ctx, keeperKey, []solana.Instruction{ix})
	if err != nil {
		// If the account already exists, Anchor returns "already in use" — treat as idempotent.
		if containsAny(err.Error(), "already in use", "custom program error: 0x0") {
			return nil
		}
		return fmt.Errorf("register wallet: %w", err)
	}
	return nil
}

func FindFaucetTrackerPDA(txlineProgram, user solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress([][]byte{[]byte("faucet_tracker"), user.Bytes()}, txlineProgram)
}

func FindUSDTTreasuryPDA(txlineProgram solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress([][]byte{[]byte("usdt_treasury")}, txlineProgram)
}

func (c *Client) RequestDevnetFaucet(ctx context.Context, user solana.PrivateKey) (solana.Signature, error) {
	faucetTracker, _, err := FindFaucetTrackerPDA(c.txlineProg, user.PublicKey())
	if err != nil {
		return solana.Signature{}, err
	}
	usdtTreasury, _, err := FindUSDTTreasuryPDA(c.txlineProg)
	if err != nil {
		return solana.Signature{}, err
	}
	userATA, _, err := solana.FindAssociatedTokenAddress(user.PublicKey(), c.mint)
	if err != nil {
		return solana.Signature{}, err
	}

	accounts := solana.AccountMetaSlice{
		solana.Meta(user.PublicKey()).SIGNER().WRITE(),
		solana.Meta(faucetTracker).WRITE(),
		solana.Meta(c.mint).WRITE(),
		solana.Meta(userATA).WRITE(),
		solana.Meta(usdtTreasury),
		solana.Meta(token.ProgramID),
		solana.Meta(associatedtokenaccount.ProgramID),
		solana.Meta(solana.SystemProgramID),
	}
	ix := solana.NewInstruction(c.txlineProg, accounts, faucetDiscriminator[:])
	_, sig, err := c.sendSigned(ctx, user, []solana.Instruction{ix})
	return sig, err
}

func (c *Client) sendSigned(ctx context.Context, payer solana.PrivateKey, instructions []solana.Instruction) (solana.PublicKey, solana.Signature, error) {
	latest, err := c.rpc.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return solana.PublicKey{}, solana.Signature{}, fmt.Errorf("latest blockhash: %w", err)
	}
	tx, err := solana.NewTransaction(instructions, latest.Value.Blockhash, solana.TransactionPayer(payer.PublicKey()))
	if err != nil {
		return solana.PublicKey{}, solana.Signature{}, err
	}
	if _, err := tx.Sign(func(pk solana.PublicKey) *solana.PrivateKey {
		if pk.Equals(payer.PublicKey()) {
			k := payer
			return &k
		}
		return nil
	}); err != nil {
		return solana.PublicKey{}, solana.Signature{}, fmt.Errorf("sign tx: %w", err)
	}

	sig, err := c.rpc.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight:       true,
		PreflightCommitment: rpc.CommitmentProcessed,
	})
	if err != nil {
		return solana.PublicKey{}, solana.Signature{}, fmt.Errorf("send tx: %w", err)
	}
	if err := waitForSignature(ctx, c.rpc, sig); err != nil {
		return solana.PublicKey{}, sig, fmt.Errorf("confirm tx %s: %w", sig, err)
	}
	return payer.PublicKey(), sig, nil
}

func (c *Client) SOLBalance(ctx context.Context, owner solana.PublicKey) (uint64, error) {
	bal, err := c.rpc.GetBalance(ctx, owner, rpc.CommitmentConfirmed)
	if err != nil {
		return 0, err
	}
	return bal.Value, nil
}

func (c *Client) TokenBalance(ctx context.Context, owner solana.PublicKey) (uint64, error) {
	ata, _, err := solana.FindAssociatedTokenAddress(owner, c.mint)
	if err != nil {
		return 0, err
	}
	bal, err := c.rpc.GetTokenAccountBalance(ctx, ata, rpc.CommitmentConfirmed)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(bal.Value.Amount, 10, 64)
}

func (c *Client) TransferSOL(ctx context.Context, from solana.PrivateKey, to solana.PublicKey, lamports uint64) (solana.Signature, error) {
	ix := solana.NewInstruction(
		solana.SystemProgramID,
		solana.AccountMetaSlice{
			solana.Meta(from.PublicKey()).SIGNER().WRITE(),
			solana.Meta(to).WRITE(),
		},
		newTransferData(lamports),
	)
	_, sig, err := c.sendSigned(ctx, from, []solana.Instruction{ix})
	return sig, err
}

func setComputeUnitLimit(units uint32) solana.Instruction {
	data := make([]byte, 5)
	data[0] = 2
	binary.LittleEndian.PutUint32(data[1:], units)
	return solana.NewInstruction(
		solana.MustPublicKeyFromBase58("ComputeBudget111111111111111111111111111111"),
		nil,
		data,
	)
}

func newTransferData(lamports uint64) []byte {
	data := make([]byte, 12)
	data[0] = 2
	data[1] = 0
	data[2] = 0
	data[3] = 0
	binary.LittleEndian.PutUint64(data[4:], lamports)
	return data
}
