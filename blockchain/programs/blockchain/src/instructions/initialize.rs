use anchor_lang::prelude::*;

use crate::{constants::*, state::*};

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(mut)]
    pub authority: Signer<'info>,

    #[account(
        init,
        payer = authority,
        space = 8 + Config::INIT_SPACE,
        seeds = [CONFIG_SEED],
        bump
    )]
    pub config: Account<'info, Config>,

    pub system_program: Program<'info, System>,
}

pub fn handle_initialize(
    ctx: Context<Initialize>,
    stablecoin_mint: Pubkey,
    txline_program: Pubkey,
) -> Result<()> {
    let config = &mut ctx.accounts.config;
    config.authority = ctx.accounts.authority.key();
    config.stablecoin_mint = stablecoin_mint;
    config.txline_program = txline_program;
    config.bump = ctx.bumps.config;

    msg!(
        "Config initialized: authority={}, mint={}, txline={}",
        config.authority,
        config.stablecoin_mint,
        config.txline_program
    );
    Ok(())
}