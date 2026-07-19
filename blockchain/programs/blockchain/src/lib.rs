pub mod constants;
pub mod cpi;
pub mod error;
pub mod instructions;
pub mod state;

use anchor_lang::prelude::*;

pub use constants::*;
pub use cpi::*;
pub use error::ErrorCode;
pub use instructions::*;
pub use state::*;

declare_id!("B39Vk22T2VPpqBEbGkW51BzFC6sNeeiQQ1mqwdvCJ2H4");

#[program]
pub mod blockchain {
    use super::*;

    pub fn initialize(
        ctx: Context<Initialize>,
        stablecoin_mint: Pubkey,
        txline_program: Pubkey,
    ) -> Result<()> {
        instructions::initialize::handle_initialize(ctx, stablecoin_mint, txline_program)
    }

    pub fn register_wallet(
        ctx: Context<RegisterWallet>,
        wallet: Pubkey,
        user_id_hash: [u8; 32],
    ) -> Result<()> {
        instructions::register_wallet::handle_register_wallet(ctx, wallet, user_id_hash)
    }

    pub fn init_match_state(ctx: Context<InitMatchState>, match_id: Vec<u8>) -> Result<()> {
        instructions::init_match_state::handle_init_match_state(ctx, match_id)
    }

    pub fn unregister_wallet(ctx: Context<UnregisterWallet>, wallet: Pubkey) -> Result<()> {
        instructions::unregister_wallet::handle_unregister_wallet(ctx, wallet)
    }

    pub fn make_wager(
        ctx: Context<MakeWager>,
        match_id: Vec<u8>,
        stake_amount: u64,
        maker_side: Side,
        invited_taker: Pubkey,
        participant1_is_home: bool,
        nonce: u64,
    ) -> Result<()> {
        instructions::make_wager::handle_make_wager(
            ctx,
            match_id,
            stake_amount,
            maker_side,
            invited_taker,
            participant1_is_home,
            nonce,
        )
    }

    pub fn accept_wager(ctx: Context<AcceptWager>, taker_side: Side) -> Result<()> {
        instructions::accept_wager::handle_accept_wager(ctx, taker_side)
    }

    pub fn close_match(ctx: Context<CloseMatch>, match_id: Vec<u8>) -> Result<()> {
        instructions::close_match::handle_close_match(ctx, match_id)
    }

    pub fn cancel_wager(ctx: Context<CancelWager>) -> Result<()> {
        instructions::cancel_wager::handle_cancel_wager(ctx)
    }

    pub fn settle_wager(
        ctx: Context<SettleWager>,
        validation: ValidateStatArgs,
        winning_side: Side,
        merkle_root: [u8; 32],
    ) -> Result<()> {
        instructions::settle_wager::handle_settle_wager(ctx, validation, winning_side, merkle_root)
    }

    pub fn void_wager(
        ctx: Context<VoidWager>,
        validation: ValidateStatArgs,
        winning_side: Side,
        merkle_root: [u8; 32],
    ) -> Result<()> {
        instructions::void_wager::handle_void_wager(ctx, validation, winning_side, merkle_root)
    }

    pub fn update_config(
        ctx: Context<UpdateConfig>,
        new_authority: Option<Pubkey>,
        new_stablecoin_mint: Option<Pubkey>,
        new_txline_program: Option<Pubkey>,
        paused: Option<bool>,
    ) -> Result<()> {
        instructions::update_config::handle_update_config(
            ctx,
            new_authority,
            new_stablecoin_mint,
            new_txline_program,
            paused,
        )
    }
}
