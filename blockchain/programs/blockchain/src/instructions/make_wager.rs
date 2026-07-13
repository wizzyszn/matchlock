use anchor_lang::prelude::*;
use anchor_spl::{
    associated_token::AssociatedToken,
    token::{self, Mint, Token, TokenAccount, TransferChecked},
};

use crate::{constants::*, error::ErrorCode, state::*};

#[derive(Accounts)]
#[instruction(match_id: Vec<u8>, stake_amount: u64, maker_side: Side, invited_taker: Pubkey, participant1_is_home: bool, nonce: u64)]
pub struct MakeWager<'info> {
    #[account(mut)]
    pub maker: Signer<'info>,

    #[account(
        seeds = [CONFIG_SEED],
        bump = config.bump,
    )]
    pub config: Box<Account<'info, Config>>,

    #[account(
        init,
        payer = maker,
        space = 8 + Wager::INIT_SPACE,
        seeds = [WAGER_SEED, maker.key().as_ref(), match_id.as_slice(), nonce.to_le_bytes().as_ref()],
        bump
    )]
    pub wager: Box<Account<'info, Wager>>,

    #[account(
        init_if_needed,
        payer = maker,
        space = 8 + MatchState::INIT_SPACE,
        seeds = [MATCH_STATE_SEED, match_id.as_slice()],
        bump,
    )]
    pub match_state: Box<Account<'info, MatchState>>,

    #[account(
        init,
        payer = maker,
        seeds = [VAULT_SEED, wager.key().as_ref()],
        bump,
        token::mint = stablecoin_mint,
        token::authority = wager,
    )]
    pub vault: Box<Account<'info, TokenAccount>>,

    #[account(
        mut,
        associated_token::mint = stablecoin_mint,
        associated_token::authority = maker,
    )]
    pub maker_stablecoin: Box<Account<'info, TokenAccount>>,

    #[account(
        constraint = stablecoin_mint.key() == config.stablecoin_mint @ ErrorCode::InvalidMint,
    )]
    pub stablecoin_mint: Box<Account<'info, Mint>>,

    #[account(
        seeds = [WALLET_PROFILE_SEED, maker.key().as_ref()],
        bump = maker_wallet_profile.bump,
        constraint = maker_wallet_profile.wallet == maker.key() @ ErrorCode::WalletNotRegistered,
    )]
    pub maker_wallet_profile: Box<Account<'info, WalletProfile>>,

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
    participant1_is_home: bool,
    nonce: u64,
) -> Result<()> {
    require!(
        !match_id.is_empty() && match_id.len() <= MAX_MATCH_ID_LEN,
        ErrorCode::InvalidMatchId,
    );
    require!(
        match_id.iter().all(|byte| byte.is_ascii_digit()),
        ErrorCode::InvalidMatchId,
    );
    require!(stake_amount > 0, ErrorCode::InvalidStake);
    require!(!ctx.accounts.config.paused, ErrorCode::ContractPaused,);
    initialize_match_state_if_needed(
        &mut ctx.accounts.match_state,
        &match_id,
        ctx.bumps.match_state,
    );
    require!(
        ctx.accounts.match_state.match_id_bytes() == match_id.as_slice(),
        ErrorCode::InvalidMatchId,
    );
    require!(!ctx.accounts.match_state.is_closed, ErrorCode::MatchClosed);
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
    wager.participant1_is_home = participant1_is_home;
    wager.maker_side = maker_side;
    wager.taker_side = Side::Unset;
    wager.stake_amount = stake_amount;
    wager.status = WagerStatus::Open;
    wager.nonce = nonce;
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

fn initialize_match_state_if_needed(
    match_state: &mut Account<MatchState>,
    match_id: &[u8],
    bump: u8,
) {
    if match_state.match_id_len != 0 {
        return;
    }
    match_state.match_id = [0u8; 32];
    match_state.match_id[..match_id.len()].copy_from_slice(match_id);
    match_state.match_id_len = match_id.len() as u8;
    match_state.is_closed = false;
    match_state.closed_at = 0;
    match_state.bump = bump;
}
