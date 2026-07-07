use anchor_lang::prelude::*;
use anchor_spl::{
    associated_token::AssociatedToken,
    token::{self, Mint, Token, TokenAccount, TransferChecked},
};

use crate::{constants::*, error::ErrorCode, state::*};

#[derive(Accounts)]
#[instruction(match_id: Vec<u8>, stake_amount: u64, maker_side: Side, invited_taker: Pubkey)]
pub struct MakeWager<'info> {
    #[account(mut)]
    pub maker: Signer<'info>,

    #[account(
        seeds = [CONFIG_SEED],
        bump = config.bump,
    )]
    pub config: Account<'info, Config>,

    #[account(
        init,
        payer = maker,
        space = 8 + Wager::INIT_SPACE,
        seeds = [WAGER_SEED, maker.key().as_ref(), match_id.as_slice()],
        bump
    )]
    pub wager: Account<'info, Wager>,

    #[account(
        init,
        payer = maker,
        seeds = [VAULT_SEED, wager.key().as_ref()],
        bump,
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
    pub system_program: Program<'info, System>,
}

pub fn handle_make_wager(
    ctx: Context<MakeWager>,
    match_id: Vec<u8>,
    stake_amount: u64,
    maker_side: Side,
    invited_taker: Pubkey,
) -> Result<()> {
    require!(
        !match_id.is_empty() && match_id.len() <= MAX_MATCH_ID_LEN,
        ErrorCode::InvalidMatchId,
    );
    require!(stake_amount > 0, ErrorCode::InvalidStake);
    if invited_taker != Pubkey::default() {
        require_keys_neq!(
            invited_taker,
            ctx.accounts.maker.key(),
            ErrorCode::CannotInviteSelf,
        );
    }

    let wager = &mut ctx.accounts.wager;
    wager.maker = ctx.accounts.maker.key();
    wager.invited_taker = invited_taker;
    wager.taker = Pubkey::default();
    wager.match_id = [0u8; 32];
    wager.match_id[..match_id.len()].copy_from_slice(&match_id);
    wager.match_id_len = match_id.len() as u8;
    wager.maker_side = maker_side;
    wager.taker_side = Side::Home;
    wager.stake_amount = stake_amount;
    wager.status = WagerStatus::Open;
    wager.bump = ctx.bumps.wager;
    wager.vault_bump = ctx.bumps.vault;

    let decimals = ctx.accounts.stablecoin_mint.decimals;

    let cpi_accounts = TransferChecked {
        from: ctx.accounts.maker_stablecoin.to_account_info(),
        mint: ctx.accounts.stablecoin_mint.to_account_info(),
        to: ctx.accounts.vault.to_account_info(),
        authority: ctx.accounts.maker.to_account_info(),
    };
    let cpi_ctx = CpiContext::new(ctx.accounts.token_program.key(), cpi_accounts);
    token::transfer_checked(cpi_ctx, stake_amount, decimals)?;

    msg!(
        "Wager opened: maker={}, stake={}, match_id_len={}, side={:?}",
        wager.maker,
        stake_amount,
        wager.match_id_len,
        maker_side
    );
    Ok(())
}