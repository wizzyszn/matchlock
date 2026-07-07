use solana_program::{
    account_info::AccountInfo, entrypoint, entrypoint::ProgramResult, program::set_return_data,
    pubkey::Pubkey,
};

solana_program::declare_id!("HusVkT78cRE5k2EgmtLJk452VxmzVjr5y6dRoaBr3oMH");

entrypoint!(process_instruction);

/// Minimal TxLINE stand-in for LiteSVM tests: accepts any instruction and returns `true`.
fn process_instruction(
    _program_id: &Pubkey,
    _accounts: &[AccountInfo],
    _instruction_data: &[u8],
) -> ProgramResult {
    set_return_data(&[1u8]);
    Ok(())
}