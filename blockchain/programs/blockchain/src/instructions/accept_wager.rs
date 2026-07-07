use anchor_lang::prelude::*;
use anchor_spl::{
    associated_token::AssociatedToken,
    token::{self, Mint, Token, TokenAccount, TransferChecked},
};

use crate::{constants::*, error::ErrorCode, state::*};

#[derive(Accounts)]
#[instruction(taker_side: Side)]
pub struct AcceptWager<'info> {
    #[account(mut)]
    pub taker: Signer<'info>,

    #[account(
        seeds = [CONFIG_SEED],
        bump = config.bump,
    )]
    pub config: Account<'info, Config>,

    #[account(
        mut,
        has_one = maker @ ErrorCode::Unauthorized,
        constraint = wager.status == WagerStatus::Open @ ErrorCode::WagerNotOpen,
    )]
    pub wager: Account<'info, Wager>,

    /// CHECK: validated via has_one on wager.maker
    pub maker: UncheckedAccount<'info>,

    #[account(
        mut,
        associated_token::mint = stablecoin_mint,
        associated_token::authority = taker,
    )]
    pub taker_stablecoin: Account<'info, TokenAccount>,

    #[account(
        mut,
        seeds = [VAULT_SEED, wager.key().as_ref()],
        bump = wager.vault_bump,
        token::mint = stablecoin_mint,
        token::authority = wager,
    )]
    pub vault: Account<'info, TokenAccount>,

    #[account(
        constraint = stablecoin_mint.key() == config.stablecoin_mint @ ErrorCode::InvalidMint,
    )]
    pub stablecoin_mint: Account<'info, Mint>,

    #[account(
        seeds = [WALLET_PROFILE_SEED, taker.key().as_ref()],
        bump = taker_wallet_profile.bump,
        constraint = taker_wallet_profile.wallet == taker.key() @ ErrorCode::WalletNotRegistered,
    )]
    pub taker_wallet_profile: Account<'info, WalletProfile>,

    pub token_program: Program<'info, Token>,
    pub associated_token_program: Program<'info, AssociatedToken>,
}

pub fn handle_accept_wager(ctx: Context<AcceptWager>, taker_side: Side) -> Result<()> {
    require_keys_neq!(
        ctx.accounts.taker.key(),
        ctx.accounts.wager.maker,
        ErrorCode::CannotAcceptOwnWager,
    );
    require!(
        taker_side != ctx.accounts.wager.maker_side,
        ErrorCode::InvalidTakerSide,
    );
    if ctx.accounts.wager.invited_taker != Pubkey::default() {
        require_keys_eq!(
            ctx.accounts.taker.key(),
            ctx.accounts.wager.invited_taker,
            ErrorCode::UnauthorizedTaker,
        );
    }

    let stake_amount = ctx.accounts.wager.stake_amount;
    let decimals = ctx.accounts.stablecoin_mint.decimals;

    let cpi_accounts = TransferChecked {
        from: ctx.accounts.taker_stablecoin.to_account_info(),
        mint: ctx.accounts.stablecoin_mint.to_account_info(),
        to: ctx.accounts.vault.to_account_info(),
        authority: ctx.accounts.taker.to_account_info(),
    };
    let cpi_ctx = CpiContext::new(ctx.accounts.token_program.key(), cpi_accounts);
    token::transfer_checked(cpi_ctx, stake_amount, decimals)?;

    let wager = &mut ctx.accounts.wager;
    wager.taker = ctx.accounts.taker.key();
    wager.taker_side = taker_side;
    wager.status = WagerStatus::Matched;

    msg!(
        "Wager matched: maker={}, taker={}, total_escrow={}",
        wager.maker,
        wager.taker,
        stake_amount
            .checked_mul(2)
            .ok_or(ProgramError::ArithmeticOverflow)?,
    );
    Ok(())
}