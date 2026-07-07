package solana

import (
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
)

const (
	ConfigSeed           = "config"
	WagerSeed            = "wager"
	VaultSeed            = "vault"
	WalletProfileSeed    = "wallet_profile"
	DailyScoresRootsSeed = "daily_scores_roots"
)

func FindWalletProfilePDA(programID, wallet solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress([][]byte{[]byte(WalletProfileSeed), wallet.Bytes()}, programID)
}

func FindConfigPDA(programID solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress([][]byte{[]byte(ConfigSeed)}, programID)
}

func FindVaultPDA(programID, wager solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress([][]byte{[]byte(VaultSeed), wager.Bytes()}, programID)
}

func FindDailyScoresRootsPDA(txlineProgram solana.PublicKey, epochDay uint16) (solana.PublicKey, uint8, error) {
	day := make([]byte, 2)
	binary.LittleEndian.PutUint16(day, epochDay)
	return solana.FindProgramAddress([][]byte{[]byte(DailyScoresRootsSeed), day}, txlineProgram)
}

func EpochDayFromMillis(ts int64) uint16 {
	const dayMs = 24 * 60 * 60 * 1000
	return uint16(ts / dayMs)
}