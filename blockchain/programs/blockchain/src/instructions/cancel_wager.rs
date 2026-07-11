use anchor_lang::prelude::*;
use anchor_spl::{
    associated_token::AssociatedToken,
    token::{self, CloseAccount, Mint, Token, TokenAccount, TransferChecked},
};

use crate::{constants::*, error::ErrorCode, state::*};

#[derive(Accounts)]
pub struct CancelWager<'info> {
    #[account(mut)]
    pub maker: Signer<'info>,

    #[account(
        seeds = [CONFIG_SEED],
        bump = config.bump,
    )]
    pub config: Account<'info, Config>,

    #[account(
        mut,
        close = maker,
        has_one = maker @ ErrorCode::Unauthorized,
        constraint = wager.status == WagerStatus::Open @ ErrorCode::InvalidStatus,
    )]
    pub wager: Account<'info, Wager>,

    #[account(
        mut,
        seeds = [VAULT_SEED, wager.key().as_ref()],
        bump = wager.vault_bump,
        token::mint = stablecoin_mint,
        token::authority = wager,
    )]
    pub vault: Account<'info, TokenAccount>,

    #[account(
        mut,
        associated_token::mint = stablecoin_mint,
        associated_token::authority = maker,
    )]
    pub maker_stablecoin: Account<'info, TokenAccount>,

    #[account(
        constraint = stablecoin_mint.key() == config.stablecoin_mint @ ErrorCode::InvalidMint,
    )]
    pub stablecoin_mint: Account<'info, Mint>,

    #[account(
        seeds = [WALLET_PROFILE_SEED, maker.key().as_ref()],
        bump = maker_wallet_profile.bump,
        constraint = maker_wallet_profile.wallet == maker.key() @ ErrorCode::WalletNotRegistered,
    )]
    pub maker_wallet_profile: Account<'info, WalletProfile>,

    pub token_program: Program<'info, Token>,
    pub associated_token_program: Program<'info, AssociatedToken>,
}

pub fn handle_cancel_wager(ctx: Context<CancelWager>) -> Result<()> {
    let wager = &ctx.accounts.wager;
    let stake_amount = wager.stake_amount;
    let decimals = ctx.accounts.stablecoin_mint.decimals;
    let bump = wager.bump;
    let maker = wager.maker;
    let match_id_len = wager.match_id_len;
    let match_id = wager.match_id;
    let match_id_slice = &match_id[..match_id_len as usize];
    let nonce = wager.nonce;

    let seeds = &[
        WAGER_SEED,
        maker.as_ref(),
        match_id_slice,
        &nonce.to_le_bytes(),
        &[bump],
    ];
    let signer = &[&seeds[..]];

    let cpi_accounts = TransferChecked {
        from: ctx.accounts.vault.to_account_info(),
        mint: ctx.accounts.stablecoin_mint.to_account_info(),
        to: ctx.accounts.maker_stablecoin.to_account_info(),
        authority: ctx.accounts.wager.to_account_info(),
    };
    let cpi_ctx =
        CpiContext::new_with_signer(ctx.accounts.token_program.key(), cpi_accounts, signer);
    token::transfer_checked(cpi_ctx, stake_amount, decimals)?;

    let close_accounts = CloseAccount {
        account: ctx.accounts.vault.to_account_info(),
        destination: ctx.accounts.maker.to_account_info(),
        authority: ctx.accounts.wager.to_account_info(),
    };
    let close_ctx =
        CpiContext::new_with_signer(ctx.accounts.token_program.key(), close_accounts, signer);
    token::close_account(close_ctx)?;

    let wager = &mut ctx.accounts.wager;
    wager.status = WagerStatus::Cancelled;

    msg!(
        "Wager cancelled: maker={}, refunded={}",
        wager.maker,
        stake_amount
    );
    Ok(())
}
