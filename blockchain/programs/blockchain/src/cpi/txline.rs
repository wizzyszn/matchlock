use anchor_lang::prelude::*;
use anchor_lang::solana_program::{
    instruction::{AccountMeta, Instruction},
    program::{get_return_data, invoke},
};

use crate::error::ErrorCode;

/// Anchor discriminator for TxLINE `validate_stat`.
pub const VALIDATE_STAT_DISCRIMINATOR: [u8; 8] = [107, 197, 232, 90, 191, 136, 105, 185];

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub struct ProofNode {
    pub hash: [u8; 32],
    pub is_right_sibling: bool,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub struct ScoresUpdateStats {
    pub update_count: i32,
    pub min_timestamp: i64,
    pub max_timestamp: i64,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub struct ScoresBatchSummary {
    pub fixture_id: i64,
    pub update_stats: ScoresUpdateStats,
    pub events_sub_tree_root: [u8; 32],
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub struct ScoreStat {
    pub key: u32,
    pub value: i32,
    pub period: i32,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub struct StatTerm {
    pub stat_to_prove: ScoreStat,
    pub event_stat_root: [u8; 32],
    pub stat_proof: Vec<ProofNode>,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub enum Comparison {
    GreaterThan,
    LessThan,
    EqualTo,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub struct TraderPredicate {
    pub threshold: i32,
    pub comparison: Comparison,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub enum BinaryExpression {
    Add,
    Subtract,
}

#[derive(AnchorSerialize, AnchorDeserialize, Clone)]
pub struct ValidateStatArgs {
    pub ts: i64,
    pub fixture_summary: ScoresBatchSummary,
    pub fixture_proof: Vec<ProofNode>,
    pub main_tree_proof: Vec<ProofNode>,
    pub predicate: TraderPredicate,
    pub stat_a: StatTerm,
    pub stat_b: Option<StatTerm>,
    pub op: Option<BinaryExpression>,
}

pub fn invoke_validate_stat<'info>(
    txline_program: &AccountInfo<'info>,
    daily_scores_merkle_roots: &AccountInfo<'info>,
    args: &ValidateStatArgs,
) -> Result<()> {
    require_keys_eq!(*txline_program.key, *daily_scores_merkle_roots.owner);

    let mut data = Vec::with_capacity(8 + 256);
    data.extend_from_slice(&VALIDATE_STAT_DISCRIMINATOR);
    args.serialize(&mut data)?;

    let ix = Instruction {
        program_id: *txline_program.key,
        accounts: vec![AccountMeta::new_readonly(
            daily_scores_merkle_roots.key(),
            false,
        )],
        data,
    };

    invoke(
        &ix,
        &[daily_scores_merkle_roots.clone(), txline_program.clone()],
    )?;

    let validated = match get_return_data() {
        Some((program_id, return_data)) => {
            require_keys_eq!(
                program_id,
                *txline_program.key,
                ErrorCode::InvalidTxlineProgram
            );
            !return_data.is_empty() && return_data[0] != 0
        }
        None => false,
    };

    require!(validated, ErrorCode::ValidationFailed);
    Ok(())
}
