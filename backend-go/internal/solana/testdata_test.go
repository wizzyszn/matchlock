package solana

import (
	"encoding/base64"
	"testing"

	"github.com/gagliardetto/solana-go"
)

func buildWagerAccountData(t *testing.T, matchID string, status uint8, stake uint64) []byte {
	t.Helper()
	data := make([]byte, wagerAccountSizeV2)
	copy(data[:8], wagerDiscriminator[:])
	maker := solana.NewWallet().PublicKey()
	taker := solana.NewWallet().PublicKey()
	copy(data[8:40], maker.Bytes())
	copy(data[72:104], taker.Bytes())
	copy(data[matchIDOffsetV2:matchIDOffsetV2+32], MatchIDFilterBytes(matchID))
	data[136] = uint8(len(matchID))
	data[137] = SideHome
	data[138] = SideAway
	for i := uint(0); i < 8; i++ {
		data[139+i] = byte(stake >> (8 * i))
	}
	data[147] = status
	data[148] = 1
	data[149] = 2
	return data
}

func encodeAccountData(data []byte) []string {
	return []string{base64.StdEncoding.EncodeToString(data), "base64"}
}