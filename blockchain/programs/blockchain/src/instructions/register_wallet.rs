use anchor_lang::prelude::*;

use crate::{constants::*, error::ErrorCode, state::*};

#[derive(Accounts)]
#[instruction(wallet: Pubkey, user_id_hash: [u8; 32])]
pub struct RegisterWallet<'info> {
    #[account(mut)]
    pub authority: Signer<'info>,

    #[account(
        seeds = [CONFIG_SEED],
        bump = config.bump,
        has_one = authority @ ErrorCode::Unauthorized,
    )]
    pub config: Account<'info, Config>,

    #[account(
        init,
        payer = authority,
        space = 8 + WalletProfile::INIT_SPACE,
        seeds = [WALLET_PROFILE_SEED, wallet.as_ref()],
        bump,
    )]
    pub wallet_profile: Account<'info, WalletProfile>,

    pub system_program: Program<'info, System>,
}

pub fn handle_register_wallet(
    ctx: Context<RegisterWallet>,
    wallet: Pubkey,
    user_id_hash: [u8; 32],
) -> Result<()> {
    require!(wallet != Pubkey::default(), ErrorCode::InvalidWallet);

    let profile = &mut ctx.accounts.wallet_profile;
    profile.wallet = wallet;
    profile.user_id_hash = user_id_hash;
    profile.bump = ctx.bumps.wallet_profile;

    msg!(
        "Wallet registered: wallet={}, user_id_hash={:?}",
        wallet,
        user_id_hash
    );
    Ok(())
}