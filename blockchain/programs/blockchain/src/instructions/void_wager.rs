use anchor_lang::prelude::*;
use anchor_spl::{
    associated_token::AssociatedToken,
    token::{self, CloseAccount, Mint, Token, TokenAccount, TransferChecked},
};

use crate::{
    constants::*,
    cpi::{invoke_validate_stat, ValidateStatArgs},
    error::ErrorCode,
    instructions::settle_wager::validate_settlement_proof,
    state::*,
};

#[derive(Accounts)]
pub struct VoidWager<'info> {
    /// Pays network fees. Must be config authority or one of the participants.
    #[account(mut)]
    pub settler: Signer<'info>,

    #[account(
        seeds = [CONFIG_SEED],
        bump = config.bump,
    )]
    pub config: Box<Account<'info, Config>>,

    #[account(
        mut,
        close = maker,
        constraint = wager.status == WagerStatus::Matched @ ErrorCode::InvalidStatus,
    )]
    pub wager: Box<Account<'info, Wager>>,

    #[account(
        mut,
        seeds = [VAULT_SEED, wager.key().as_ref()],
        bump = wager.vault_bump,
        token::mint = stablecoin_mint,
        token::authority = wager,
    )]
    pub vault: Box<Account<'info, TokenAccount>>,

    /// CHECK: Address is constrained to the wager maker.
    #[account(
        mut,
        constraint = maker.key() == wager.maker @ ErrorCode::Unauthorized,
    )]
    pub maker: UncheckedAccount<'info>,

    #[account(
        mut,
        associated_token::mint = stablecoin_mint,
        associated_token::authority = maker,
    )]
    pub maker_stablecoin: Box<Account<'info, TokenAccount>>,

    /// CHECK: Address is constrained to the wager taker.
    #[account(
        mut,
        constraint = taker.key() == wager.taker @ ErrorCode::Unauthorized,
    )]
    pub taker: UncheckedAccount<'info>,

    #[account(
        mut,
        associated_token::mint = stablecoin_mint,
        associated_token::authority = taker,
    )]
    pub taker_stablecoin: Box<Account<'info, TokenAccount>>,

    #[account(
        constraint = stablecoin_mint.key() == config.stablecoin_mint @ ErrorCode::InvalidMint,
    )]
    pub stablecoin_mint: Box<Account<'info, Mint>>,

    /// CHECK: TxLINE daily scores Merkle roots PDA; owner must be config.txline_program.
    #[account(
        constraint = daily_scores_merkle_roots.owner == &config.txline_program @ ErrorCode::InvalidTxlineProgram,
    )]
    pub daily_scores_merkle_roots: UncheckedAccount<'info>,

    /// CHECK: TxLINE program; must match config and be executable for CPI.
    #[account(
        constraint = txline_program.key() == config.txline_program @ ErrorCode::InvalidTxlineProgram,
        constraint = txline_program.executable @ ErrorCode::InvalidTxlineProgram,
    )]
    pub txline_program: UncheckedAccount<'info>,

    pub token_program: Program<'info, Token>,
    pub associated_token_program: Program<'info, AssociatedToken>,
}

pub fn handle_void_wager(
    ctx: Context<VoidWager>,
    validation: ValidateStatArgs,
    winning_side: Side,
    merkle_root: [u8; 32],
) -> Result<()> {
    let wager = &ctx.accounts.wager;
    require!(
        winning_side != Side::Unset
            && winning_side != wager.maker_side
            && winning_side != wager.taker_side,
        ErrorCode::InvalidVoidOutcome,
    );

    let settler = ctx.accounts.settler.key();
    require!(
        settler == ctx.accounts.config.authority
            || settler == wager.maker
            || settler == wager.taker,
        ErrorCode::Unauthorized,
    );

    validate_settlement_proof(wager, &validation, winning_side, merkle_root)?;
    invoke_validate_stat(
        &ctx.accounts.txline_program.to_account_info(),
        &ctx.accounts.daily_scores_merkle_roots.to_account_info(),
        &validation,
    )?;

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

    let maker_transfer = TransferChecked {
        from: ctx.accounts.vault.to_account_info(),
        mint: ctx.accounts.stablecoin_mint.to_account_info(),
        to: ctx.accounts.maker_stablecoin.to_account_info(),
        authority: ctx.accounts.wager.to_account_info(),
    };
    token::transfer_checked(
        CpiContext::new_with_signer(ctx.accounts.token_program.key(), maker_transfer, signer),
        stake_amount,
        decimals,
    )?;

    let taker_transfer = TransferChecked {
        from: ctx.accounts.vault.to_account_info(),
        mint: ctx.accounts.stablecoin_mint.to_account_info(),
        to: ctx.accounts.taker_stablecoin.to_account_info(),
        authority: ctx.accounts.wager.to_account_info(),
    };
    token::transfer_checked(
        CpiContext::new_with_signer(ctx.accounts.token_program.key(), taker_transfer, signer),
        stake_amount,
        decimals,
    )?;

    let close_accounts = CloseAccount {
        account: ctx.accounts.vault.to_account_info(),
        destination: ctx.accounts.maker.to_account_info(),
        authority: ctx.accounts.wager.to_account_info(),
    };
    token::close_account(CpiContext::new_with_signer(
        ctx.accounts.token_program.key(),
        close_accounts,
        signer,
    ))?;

    let wager = &mut ctx.accounts.wager;
    wager.status = WagerStatus::Cancelled;
    msg!(
        "Wager voided: maker={}, taker={}, refunded_each={}, outcome={:?}",
        wager.maker,
        wager.taker,
        stake_amount,
        winning_side,
    );
    Ok(())
}
