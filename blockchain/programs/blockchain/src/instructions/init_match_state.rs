use anchor_lang::prelude::*;

use crate::{constants::*, error::ErrorCode, state::*};

#[derive(Accounts)]
#[instruction(match_id: Vec<u8>)]
pub struct InitMatchState<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,

    #[account(
        init,
        payer = payer,
        space = 8 + MatchState::INIT_SPACE,
        seeds = [MATCH_STATE_SEED, match_id.as_slice()],
        bump
    )]
    pub match_state: Account<'info, MatchState>,

    pub system_program: Program<'info, System>,
}

pub fn handle_init_match_state(ctx: Context<InitMatchState>, match_id: Vec<u8>) -> Result<()> {
    require!(
        !match_id.is_empty() && match_id.len() <= MAX_MATCH_ID_LEN,
        ErrorCode::InvalidMatchId,
    );
    require!(
        match_id.iter().all(|byte| byte.is_ascii_digit()),
        ErrorCode::InvalidMatchId,
    );

    let match_state = &mut ctx.accounts.match_state;
    match_state.match_id = [0u8; 32];
    match_state.match_id[..match_id.len()].copy_from_slice(&match_id);
    match_state.match_id_len = match_id.len() as u8;
    match_state.is_closed = false;
    match_state.closed_at = 0;
    match_state.bump = ctx.bumps.match_state;
    Ok(())
}
