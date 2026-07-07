mod common;

use anchor_lang::{InstructionData, ToAccountMetas};
use blockchain::state::Side;
use common::{TestEnv, ATA_PROGRAM_ID, TXLINE_MOCK_PROGRAM_ID};
use solana_instruction::Instruction;
use solana_signer::Signer;
use solana_transaction::Transaction;
use spl_associated_token_account_interface::address::get_associated_token_address;
use spl_token_interface::ID as TOKEN_PROGRAM_ID;

#[test]
fn unregistered_wallet_cannot_make_wager() {
    let mut env = TestEnv::new();
    let stranger = solana_keypair::Keypair::new();
    env.svm.airdrop(&stranger.pubkey(), 10_000_000_000).unwrap();
    env.fund_wallet(&stranger.pubkey(), 5_000_000);

    let maker = stranger.pubkey();
    let (wager, vault) = env.wager_pdas(&maker, &env.match_id);
    let maker_ata = get_associated_token_address(&maker, &env.mint);
    let profile = env.wallet_profile_pda(&maker);

    let ix = Instruction::new_with_bytes(
        blockchain::id(),
        &blockchain::instruction::MakeWager {
            match_id: env.match_id.clone(),
            stake_amount: 100_000,
            maker_side: Side::Home,
            invited_taker: solana_address::Address::default(),
        }
        .data(),
        blockchain::accounts::MakeWager {
            maker,
            config: env.config,
            wager,
            vault,
            maker_stablecoin: maker_ata,
            stablecoin_mint: env.mint,
            maker_wallet_profile: profile,
            token_program: TOKEN_PROGRAM_ID,
            associated_token_program: ATA_PROGRAM_ID,
            system_program: solana_system_interface::program::ID,
        }
        .to_account_metas(None),
    );

    let tx = Transaction::new_signed_with_payer(
        &[ix],
        Some(&maker),
        &[&stranger],
        env.svm.latest_blockhash(),
    );
    assert!(env.svm.send_transaction(tx).is_err());
}

#[test]
fn make_and_accept_wager_happy_path() {
    let mut env = TestEnv::new();
    let stake = 1_000_000u64;
    let maker_ata = get_associated_token_address(&env.maker.pubkey(), &env.mint);
    let taker_ata = get_associated_token_address(&env.taker.pubkey(), &env.mint);
    let maker_before = env.token_balance(&maker_ata);

    let (wager, vault) = env.make_wager(stake, Side::Home);
    assert_eq!(env.token_balance(&maker_ata), maker_before - stake);
    assert_eq!(env.token_balance(&vault), stake);

    let taker_before = env.token_balance(&taker_ata);
    env.accept_wager(wager, vault, Side::Away);
    assert_eq!(env.token_balance(&taker_ata), taker_before - stake);
    assert_eq!(env.token_balance(&vault), stake * 2);
}

#[test]
fn unregistered_wallet_cannot_accept_wager() {
    let mut env = TestEnv::new();
    let stranger = solana_keypair::Keypair::new();
    env.svm.airdrop(&stranger.pubkey(), 10_000_000_000).unwrap();
    env.fund_wallet(&stranger.pubkey(), 5_000_000);

    let (wager, vault) = env.make_wager(100_000, Side::Home);
    let stranger_ata = get_associated_token_address(&stranger.pubkey(), &env.mint);

    let ix = Instruction::new_with_bytes(
        blockchain::id(),
        &blockchain::instruction::AcceptWager {
            taker_side: Side::Away,
        }
        .data(),
        blockchain::accounts::AcceptWager {
            taker: stranger.pubkey(),
            config: env.config,
            wager,
            maker: env.maker.pubkey(),
            taker_stablecoin: stranger_ata,
            vault,
            stablecoin_mint: env.mint,
            taker_wallet_profile: env.wallet_profile_pda(&stranger.pubkey()),
            token_program: TOKEN_PROGRAM_ID,
            associated_token_program: ATA_PROGRAM_ID,
        }
        .to_account_metas(None),
    );

    let tx = Transaction::new_signed_with_payer(
        &[ix],
        Some(&stranger.pubkey()),
        &[&stranger],
        env.svm.latest_blockhash(),
    );
    assert!(env.svm.send_transaction(tx).is_err());
}

#[test]
fn cancel_wager_refunds_maker() {
    let mut env = TestEnv::new();
    let stake = 500_000u64;
    let maker_ata = get_associated_token_address(&env.maker.pubkey(), &env.mint);
    let before = env.token_balance(&maker_ata);

    let (wager, vault) = env.make_wager(stake, Side::Away);
    env.cancel_wager(wager, vault);

    assert_eq!(env.token_balance(&maker_ata), before);
    assert!(env.svm.get_account(&wager).is_none());
    assert!(env.svm.get_account(&vault).is_none());
}

#[test]
fn accept_own_wager_fails() {
    let mut env = TestEnv::new();
    let (wager, vault) = env.make_wager(100_000, Side::Home);
    let maker = env.maker.pubkey();
    let maker_ata = get_associated_token_address(&maker, &env.mint);

    let ix = Instruction::new_with_bytes(
        blockchain::id(),
        &blockchain::instruction::AcceptWager {
            taker_side: Side::Away,
        }
        .data(),
        blockchain::accounts::AcceptWager {
            taker: maker,
            config: env.config,
            wager,
            maker,
            taker_stablecoin: maker_ata,
            vault,
            stablecoin_mint: env.mint,
            taker_wallet_profile: env.wallet_profile_pda(&maker),
            token_program: TOKEN_PROGRAM_ID,
            associated_token_program: ATA_PROGRAM_ID,
        }
        .to_account_metas(None),
    );

    let tx = Transaction::new_signed_with_payer(
        &[ix],
        Some(&maker),
        &[&env.maker],
        env.svm.latest_blockhash(),
    );
    assert!(env.svm.send_transaction(tx).is_err());
}

#[test]
fn cancel_matched_wager_fails() {
    let mut env = TestEnv::new();
    let (wager, vault) = env.make_wager(250_000, Side::Home);
    env.accept_wager(wager, vault, Side::Away);

    let maker = env.maker.pubkey();
    let maker_ata = get_associated_token_address(&maker, &env.mint);
    let ix = Instruction::new_with_bytes(
        blockchain::id(),
        &blockchain::instruction::CancelWager {}.data(),
        blockchain::accounts::CancelWager {
            maker,
            config: env.config,
            wager,
            vault,
            maker_stablecoin: maker_ata,
            stablecoin_mint: env.mint,
            maker_wallet_profile: env.wallet_profile_pda(&maker),
            token_program: TOKEN_PROGRAM_ID,
            associated_token_program: ATA_PROGRAM_ID,
        }
        .to_account_metas(None),
    );

    let tx = Transaction::new_signed_with_payer(
        &[ix],
        Some(&maker),
        &[&env.maker],
        env.svm.latest_blockhash(),
    );
    assert!(env.svm.send_transaction(tx).is_err());
}

#[test]
fn settle_wager_pays_winner() {
    let mut env = TestEnv::new();
    let stake = 750_000u64;
    let (wager, vault) = env.make_wager(stake, Side::Home);
    env.accept_wager(wager, vault, Side::Away);

    let maker_ata = get_associated_token_address(&env.maker.pubkey(), &env.mint);
    let maker_before = env.token_balance(&maker_ata);

    let maker = env.maker.insecure_clone();
    env.settle_wager(wager, vault, Side::Home, &maker);

    assert_eq!(env.token_balance(&maker_ata), maker_before + stake * 2);
    assert!(env.svm.get_account(&wager).is_none());
    assert!(env.svm.get_account(&vault).is_none());
}

#[test]
fn settle_wager_keeper_crank_still_works() {
    let mut env = TestEnv::new();
    let stake = 500_000u64;
    let (wager, vault) = env.make_wager(stake, Side::Away);
    env.accept_wager(wager, vault, Side::Home);

    let maker_ata = get_associated_token_address(&env.maker.pubkey(), &env.mint);
    let maker_before = env.token_balance(&maker_ata);
    env.settle_wager_as_keeper(wager, vault, Side::Away, env.maker.pubkey());

    assert_eq!(env.token_balance(&maker_ata), maker_before + stake * 2);
}

#[test]
fn settle_wager_wrong_winner_side_fails() {
    let mut env = TestEnv::new();
    let (wager, vault) = env.make_wager(100_000, Side::Home);
    env.accept_wager(wager, vault, Side::Away);

    let maker = env.maker.pubkey();
    let maker_ata = get_associated_token_address(&maker, &env.mint);
    let daily_scores = env.setup_daily_scores_account();

    let ix = Instruction::new_with_bytes(
        blockchain::id(),
        &blockchain::instruction::SettleWager {
            validation: env.minimal_validation(),
            winning_side: Side::Away,
            merkle_root: [1u8; 32],
        }
        .data(),
        blockchain::accounts::SettleWager {
            settler: env.authority.pubkey(),
            config: env.config,
            wager,
            vault,
            winner: maker,
            winner_stablecoin: maker_ata,
            stablecoin_mint: env.mint,
            daily_scores_merkle_roots: daily_scores,
            txline_program: TXLINE_MOCK_PROGRAM_ID,
            token_program: TOKEN_PROGRAM_ID,
            associated_token_program: ATA_PROGRAM_ID,
        }
        .to_account_metas(None),
    );

    let tx = Transaction::new_signed_with_payer(
        &[ix],
        Some(&env.authority.pubkey()),
        &[&env.authority],
        env.svm.latest_blockhash(),
    );  
    assert!(env.svm.send_transaction(tx).is_err());
}

#[test]
fn draw_wager_settles_for_draw_backer() {
    let mut env = TestEnv::new();
    let stake = 400_000u64;
    let (wager, vault) = env.make_wager(stake, Side::Draw);
    env.accept_wager(wager, vault, Side::Home);

    let maker_ata = get_associated_token_address(&env.maker.pubkey(), &env.mint);
    let maker_before = env.token_balance(&maker_ata);
    let maker = env.maker.insecure_clone();
    env.settle_wager(wager, vault, Side::Draw, &maker);

    assert_eq!(env.token_balance(&maker_ata), maker_before + stake * 2);
}

#[test]
fn accept_same_side_as_maker_fails() {
    let mut env = TestEnv::new();
    let (wager, vault) = env.make_wager(100_000, Side::Draw);
    let taker = env.taker.pubkey();
    let taker_ata = get_associated_token_address(&taker, &env.mint);
    let maker = env.maker.pubkey();

    let ix = Instruction::new_with_bytes(
        blockchain::id(),
        &blockchain::instruction::AcceptWager {
            taker_side: Side::Draw,
        }
        .data(),
        blockchain::accounts::AcceptWager {
            taker,
            config: env.config,
            wager,
            maker,
            taker_stablecoin: taker_ata,
            vault,
            stablecoin_mint: env.mint,
            taker_wallet_profile: env.wallet_profile_pda(&taker),
            token_program: TOKEN_PROGRAM_ID,
            associated_token_program: ATA_PROGRAM_ID,
        }
        .to_account_metas(None),
    );

    let tx = Transaction::new_signed_with_payer(
        &[ix],
        Some(&taker),
        &[&env.taker],
        env.svm.latest_blockhash(),
    );
    assert!(env.svm.send_transaction(tx).is_err());
}

#[test]
fn invited_taker_can_accept_direct_challenge() {
    let mut env = TestEnv::new();
    let invited = env.taker.pubkey();
    let (wager, vault) = env.make_wager_for(250_000, Side::Home, invited);
    env.accept_wager(wager, vault, Side::Away);
}

#[test]
fn wrong_wallet_cannot_accept_invited_wager() {
    let mut env = TestEnv::new();
    let stranger = solana_keypair::Keypair::new();
    env.svm.airdrop(&stranger.pubkey(), 10_000_000_000).unwrap();
    env.register_wallet(&stranger.pubkey(), [3u8; 32]);
    env.fund_wallet(&stranger.pubkey(), 5_000_000);

    let invited = env.taker.pubkey();
    let (wager, vault) = env.make_wager_for(100_000, Side::Home, invited);

    let stranger_ata = get_associated_token_address(&stranger.pubkey(), &env.mint);
    let ix = Instruction::new_with_bytes(
        blockchain::id(),
        &blockchain::instruction::AcceptWager {
            taker_side: Side::Away,
        }
        .data(),
        blockchain::accounts::AcceptWager {
            taker: stranger.pubkey(),
            config: env.config,
            wager,
            maker: env.maker.pubkey(),
            taker_stablecoin: stranger_ata,
            vault,
            stablecoin_mint: env.mint,
            taker_wallet_profile: env.wallet_profile_pda(&stranger.pubkey()),
            token_program: TOKEN_PROGRAM_ID,
            associated_token_program: ATA_PROGRAM_ID,
        }
        .to_account_metas(None),
    );
    let tx = Transaction::new_signed_with_payer(
        &[ix],
        Some(&stranger.pubkey()),
        &[&stranger],
        env.svm.latest_blockhash(),
    );
    assert!(env.svm.send_transaction(tx).is_err());
}
