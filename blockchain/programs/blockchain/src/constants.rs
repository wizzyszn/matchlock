use anchor_lang::prelude::*;

#[constant]
pub const CONFIG_SEED: &[u8] = b"config";

#[constant]
pub const WAGER_SEED: &[u8] = b"wager";

#[constant]
pub const VAULT_SEED: &[u8] = b"vault";

#[constant]
pub const WALLET_PROFILE_SEED: &[u8] = b"wallet_profile";

#[constant]
pub const MATCH_STATE_SEED: &[u8] = b"match_state";

pub const MAX_MATCH_ID_LEN: usize = 32;

/// TxLINE devnet program ID (https://txline-docs.txodds.com/documentation/programs/addresses)
pub const TXLINE_DEVNET_PROGRAM_ID: Pubkey =
    pubkey!("6pW64gN1s2uqjHkn1unFeEjAwJkPGHoppGvS715wyP2J");

/// TxLINE devnet USDT mint used as wager stablecoin on devnet.
pub const DEVNET_STABLECOIN_MINT: Pubkey = pubkey!("ELWTKspHKCnCfCiCiqYw1EDH77k8VCP74dK9qytG2Ujh");
