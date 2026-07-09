use anchor_lang::prelude::*;

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, PartialEq, Eq, Debug, InitSpace)]
pub enum WagerStatus {
    Open,
    Matched,
    Settled,
    Cancelled,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, PartialEq, Eq, Debug, InitSpace)]
pub enum Side {
    Home,
    Away,
    Draw,
}

/// WalletProfile binds a Solana wallet to a Matchlock platform user (one wallet → one user).
/// Created by config authority after off-chain wallet link verification.
#[account]
#[derive(InitSpace)]
pub struct WalletProfile {
    pub wallet: Pubkey,
    pub user_id_hash: [u8; 32],
    pub bump: u8,
}

#[account]
#[derive(InitSpace)]
pub struct Config {
    pub authority: Pubkey,
    pub stablecoin_mint: Pubkey,
    pub txline_program: Pubkey,
    pub bump: u8,
}

#[account]
#[derive(InitSpace)]
pub struct Wager {
    pub maker: Pubkey,
    /// When set (not default), only this pubkey may accept the open wager.
    pub invited_taker: Pubkey,
    pub taker: Pubkey,
    pub match_id: [u8; 32],
    pub match_id_len: u8,
    pub maker_side: Side,
    /// Set on accept; meaningful once status is Matched or later.
    pub taker_side: Side,
    pub stake_amount: u64,
    pub status: WagerStatus,
    pub bump: u8,
    pub vault_bump: u8,
}

impl Wager {
    pub fn match_id_bytes(&self) -> &[u8] {
        &self.match_id[..self.match_id_len as usize]
    }
}
