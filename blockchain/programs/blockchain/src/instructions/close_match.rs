use anchor_lang::prelude::*;

use crate::{constants::*, error::ErrorCode, state::*};

#[derive(Accounts)]
#[instruction(match_id: Vec<u8>)]
pub struct CloseMatch<'info> {
    #[account(mut)]
    pub authority: Signer<'info>,

    #[account(
        seeds = [CONFIG_SEED],
        bump = config.bump,
        has_one = authority @ ErrorCode::Unauthorized,
    )]
    pub config: Box<Account<'info, Config>>,

    #[account(
        init_if_needed,
        payer = authority,
        space = 8 + MatchState::INIT_SPACE,
        seeds = [MATCH_STATE_SEED, match_id.as_slice()],
        bump,
    )]
    pub match_state: Box<Account<'info, MatchState>>,

    pub system_program: Program<'info, System>,
}

pub fn handle_close_match(ctx: Context<CloseMatch>, match_id: Vec<u8>) -> Result<()> {
    require!(
        !match_id.is_empty() && match_id.len() <= MAX_MATCH_ID_LEN,
        ErrorCode::InvalidMatchId,
    );
    require!(
        match_id.iter().all(|byte| byte.is_ascii_digit()),
        ErrorCode::InvalidMatchId,
    );

    let match_state = &mut ctx.accounts.match_state;
    if match_state.match_id_len == 0 {
        match_state.match_id = [0u8; 32];
        match_state.match_id[..match_id.len()].copy_from_slice(&match_id);
        match_state.match_id_len = match_id.len() as u8;
        match_state.bump = ctx.bumps.match_state;
    }
    require!(
        match_state.match_id_bytes() == match_id.as_slice(),
        ErrorCode::InvalidMatchId,
    );
    if match_state.is_closed {
        return Ok(());
    }
    match_state.is_closed = true;
    match_state.closed_at = Clock::get()?.unix_timestamp;

    msg!(
        "Match closed for wagering: len={}",
        match_state.match_id_len
    );
    Ok(())
}
