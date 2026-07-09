use anchor_lang::prelude::*;

#[error_code]
pub enum ErrorCode {
    #[msg("Unauthorized signer for this wager action")]
    Unauthorized,
    #[msg("Wager is not in the required status for this action")]
    InvalidStatus,
    #[msg("Match ID must be between 1 and 32 bytes")]
    InvalidMatchId,
    #[msg("Stake amount must be greater than zero")]
    InvalidStake,
    #[msg("Maker cannot accept their own wager")]
    CannotAcceptOwnWager,
    #[msg("Only open wagers can be accepted")]
    WagerNotOpen,
    #[msg("Stablecoin mint does not match program config")]
    InvalidMint,
    #[msg("TxLINE program ID does not match program config")]
    InvalidTxlineProgram,
    #[msg("Taker must pick a different outcome than the maker")]
    InvalidTakerSide,
    #[msg("Winning side must match maker or taker position")]
    InvalidWinningSide,
    #[msg("TxLINE stat validation failed or returned false")]
    ValidationFailed,
    #[msg("Wager is already settled")]
    AlreadySettled,
    #[msg("Maker cannot invite themselves as taker")]
    CannotInviteSelf,
    #[msg("Only the invited wallet may accept this wager")]
    UnauthorizedTaker,
    #[msg("Wallet is not registered with Matchlock")]
    WalletNotRegistered,
    #[msg("Wallet is already registered to another account")]
    WalletAlreadyRegistered,
    #[msg("Invalid wallet pubkey")]
    InvalidWallet,
}
