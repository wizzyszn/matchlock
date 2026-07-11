use anchor_lang::prelude::*;

use crate::{constants::*, error::ErrorCode, state::*};

#[derive(Accounts)]
pub struct UpdateConfig<'info> {
    #[account(mut)]
    pub authority: Signer<'info>,

    #[account(
        mut,
        seeds = [CONFIG_SEED],
        bump = config.bump,
        has_one = authority @ ErrorCode::Unauthorized,
    )]
    pub config: Account<'info, Config>,
}

pub fn handle_update_config(
    ctx: Context<UpdateConfig>,
    new_authority: Option<Pubkey>,
    new_stablecoin_mint: Option<Pubkey>,
    new_txline_program: Option<Pubkey>,
    paused: Option<bool>,
) -> Result<()> {
    let config = &mut ctx.accounts.config;

    if let Some(authority) = new_authority {
        require!(authority != Pubkey::default(), ErrorCode::Unauthorized);
        config.authority = authority;
    }
    if let Some(mint) = new_stablecoin_mint {
        config.stablecoin_mint = mint;
    }
    if let Some(program) = new_txline_program {
        config.txline_program = program;
    }
    if let Some(p) = paused {
        config.paused = p;
    }

    msg!(
        "Config updated: authority={}, paused={}",
        config.authority,
        config.paused
    );
    Ok(())
}
