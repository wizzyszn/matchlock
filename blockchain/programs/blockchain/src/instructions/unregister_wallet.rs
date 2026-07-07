use anchor_lang::prelude::*;

use crate::{constants::*, error::ErrorCode, state::*};

#[derive(Accounts)]
#[instruction(wallet: Pubkey)]
pub struct UnregisterWallet<'info> {
    #[account(mut)]
    pub authority: Signer<'info>,

    #[account(
        seeds = [CONFIG_SEED],
        bump = config.bump,
        has_one = authority @ ErrorCode::Unauthorized,
    )]
    pub config: Account<'info, Config>,

    #[account(
        mut,
        close = authority,
        seeds = [WALLET_PROFILE_SEED, wallet.as_ref()],
        bump = wallet_profile.bump,
        constraint = wallet_profile.wallet == wallet @ ErrorCode::InvalidWallet,
    )]
    pub wallet_profile: Account<'info, WalletProfile>,
}

pub fn handle_unregister_wallet(_ctx: Context<UnregisterWallet>, wallet: Pubkey) -> Result<()> {
    require!(wallet != Pubkey::default(), ErrorCode::InvalidWallet);
    msg!("Wallet unregistered: wallet={}", wallet);
    Ok(())
}
