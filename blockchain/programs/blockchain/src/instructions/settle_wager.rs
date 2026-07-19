use anchor_lang::prelude::*;
use anchor_spl::{
    associated_token::AssociatedToken,
    token::{self, CloseAccount, Mint, Token, TokenAccount, TransferChecked},
};

use crate::{
    constants::*,
    cpi::{invoke_validate_stat, Comparison, ValidateStatArgs},
    error::ErrorCode,
    state::*,
};

pub(crate) const TXLINE_STAT_DRAW: u32 = 1001;
pub(crate) const TXLINE_STAT_P1_WIN: u32 = 1002;
pub(crate) const TXLINE_STAT_P2_WIN: u32 = 1003;

#[derive(Accounts)]
pub struct SettleWager<'info> {
    /// Pays network fees. Must be config authority (keeper crank) or the winning participant.
    #[account(mut)]
    pub settler: Signer<'info>,

    #[account(
        seeds = [CONFIG_SEED],
        bump = config.bump,
    )]
    pub config: Account<'info, Config>,

    #[account(
        mut,
        close = winner,
        constraint = wager.status == WagerStatus::Matched @ ErrorCode::InvalidStatus,
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

    /// CHECK: Must be maker or taker; receives closed account rent.
    #[account(
        mut,
        constraint = winner.key() == wager.maker || winner.key() == wager.taker @ ErrorCode::Unauthorized,
    )]
    pub winner: UncheckedAccount<'info>,

    #[account(
        mut,
        associated_token::mint = stablecoin_mint,
        associated_token::authority = winner,
    )]
    pub winner_stablecoin: Account<'info, TokenAccount>,

    #[account(
        constraint = stablecoin_mint.key() == config.stablecoin_mint @ ErrorCode::InvalidMint,
    )]
    pub stablecoin_mint: Account<'info, Mint>,

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

pub fn handle_settle_wager(
    ctx: Context<SettleWager>,
    validation: ValidateStatArgs,
    winning_side: Side,
    merkle_root: [u8; 32],
) -> Result<()> {
    let wager = &ctx.accounts.wager;

    let expected_winner = if winning_side == wager.maker_side {
        wager.maker
    } else if winning_side == wager.taker_side {
        wager.taker
    } else {
        return Err(ErrorCode::InvalidWinningSide.into());
    };

    require_keys_eq!(
        ctx.accounts.winner.key(),
        expected_winner,
        ErrorCode::InvalidWinningSide,
    );

    let settler = ctx.accounts.settler.key();
    require!(
        settler == ctx.accounts.config.authority || settler == expected_winner,
        ErrorCode::Unauthorized,
    );

    validate_settlement_proof(wager, &validation, winning_side, merkle_root)?;

    invoke_validate_stat(
        &ctx.accounts.txline_program.to_account_info(),
        &ctx.accounts.daily_scores_merkle_roots.to_account_info(),
        &validation,
    )?;

    let total_payout = wager
        .stake_amount
        .checked_mul(2)
        .ok_or(ProgramError::ArithmeticOverflow)?;

    let bump = wager.bump;
    let maker = wager.maker;
    let match_id_len = wager.match_id_len;
    let match_id = wager.match_id;
    let match_id_slice = &match_id[..match_id_len as usize];
    let nonce = wager.nonce;
    let decimals = ctx.accounts.stablecoin_mint.decimals;

    let seeds = &[
        WAGER_SEED,
        maker.as_ref(),
        match_id_slice,
        &nonce.to_le_bytes(),
        &[bump],
    ];
    let signer = &[&seeds[..]];

    let transfer_accounts = TransferChecked {
        from: ctx.accounts.vault.to_account_info(),
        mint: ctx.accounts.stablecoin_mint.to_account_info(),
        to: ctx.accounts.winner_stablecoin.to_account_info(),
        authority: ctx.accounts.wager.to_account_info(),
    };
    let transfer_ctx =
        CpiContext::new_with_signer(ctx.accounts.token_program.key(), transfer_accounts, signer);
    token::transfer_checked(transfer_ctx, total_payout, decimals)?;

    let close_accounts = CloseAccount {
        account: ctx.accounts.vault.to_account_info(),
        destination: ctx.accounts.winner.to_account_info(),
        authority: ctx.accounts.wager.to_account_info(),
    };
    let close_ctx =
        CpiContext::new_with_signer(ctx.accounts.token_program.key(), close_accounts, signer);
    token::close_account(close_ctx)?;

    let wager = &mut ctx.accounts.wager;
    wager.status = WagerStatus::Settled;

    msg!("match_id_len={}", wager.match_id_len);
    msg!("merkle_root={:?}", merkle_root);
    msg!("winner={}", expected_winner);
    msg!("stake={}", wager.stake_amount);
    msg!(
        "Wager settled: maker={}, taker={}, payout={}",
        wager.maker,
        wager.taker,
        total_payout
    );
    Ok(())
}

pub(crate) fn validate_settlement_proof(
    wager: &Wager,
    validation: &ValidateStatArgs,
    winning_side: Side,
    merkle_root: [u8; 32],
) -> Result<()> {
    require!(
        validation.fixture_summary.fixture_id == wager_fixture_id(wager)?,
        ErrorCode::InvalidSettlementProof,
    );
    require!(
        validation.fixture_summary.events_sub_tree_root == merkle_root,
        ErrorCode::InvalidSettlementProof,
    );
    require!(
        validation.predicate.threshold == 0
            && validation.predicate.comparison == Comparison::GreaterThan
            && validation.stat_b.is_none()
            && validation.op.is_none(),
        ErrorCode::InvalidSettlementProof,
    );
    require!(
        validation.stat_a.stat_to_prove.value > 0,
        ErrorCode::ValidationFailed,
    );

    let stat_key = validation.stat_a.stat_to_prove.key;
    let expected_stat_key = stat_key_for_winning_side(winning_side, wager.participant1_is_home)?;
    require!(
        stat_key == expected_stat_key,
        ErrorCode::InvalidSettlementProof,
    );
    Ok(())
}

pub(crate) fn stat_key_for_winning_side(
    winning_side: Side,
    participant1_is_home: bool,
) -> Result<u32> {
    match winning_side {
        Side::Draw => Ok(TXLINE_STAT_DRAW),
        Side::Home => {
            if participant1_is_home {
                Ok(TXLINE_STAT_P1_WIN)
            } else {
                Ok(TXLINE_STAT_P2_WIN)
            }
        }
        Side::Away => {
            if participant1_is_home {
                Ok(TXLINE_STAT_P2_WIN)
            } else {
                Ok(TXLINE_STAT_P1_WIN)
            }
        }
        Side::Unset => Err(ErrorCode::InvalidWinningSide.into()),
    }
}

pub(crate) fn wager_fixture_id(wager: &Wager) -> Result<i64> {
    let mut out = 0i64;
    for byte in wager.match_id_bytes() {
        require!(byte.is_ascii_digit(), ErrorCode::InvalidMatchId);
        let digit = (*byte - b'0') as i64;
        out = out
            .checked_mul(10)
            .and_then(|value| value.checked_add(digit))
            .ok_or(ErrorCode::InvalidMatchId)?;
    }
    Ok(out)
}
