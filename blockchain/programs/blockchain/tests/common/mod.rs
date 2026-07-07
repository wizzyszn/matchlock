use anchor_lang::{AccountDeserialize, InstructionData, ToAccountMetas};
use blockchain::{
    constants::{CONFIG_SEED, VAULT_SEED, WAGER_SEED, WALLET_PROFILE_SEED},
    cpi::ValidateStatArgs,
    state::{Config, Side, Wager, WagerStatus, WalletProfile},
};
use litesvm::LiteSVM;
use solana_account::Account;
use solana_address::Address;
use solana_instruction::Instruction;
use solana_keypair::Keypair;
use solana_program_pack::Pack;
use solana_signer::Signer;
use solana_system_interface::instruction::create_account;
use solana_transaction::Transaction;
use spl_associated_token_account_interface::{
    address::get_associated_token_address, instruction::create_associated_token_account,
    program::id as ata_program_id,
};
use spl_token_interface::{
    instruction::{initialize_mint2, mint_to},
    state::Account as TokenAccountState,
    ID as TOKEN_PROGRAM_ID,
};

pub const ATA_PROGRAM_ID: Address = ata_program_id();

pub const TXLINE_MOCK_PROGRAM_ID: Address =
    solana_address::address!("HusVkT78cRE5k2EgmtLJk452VxmzVjr5y6dRoaBr3oMH");

pub struct TestEnv {
    pub svm: LiteSVM,
    pub authority: Keypair,
    pub maker: Keypair,
    pub taker: Keypair,
    pub mint: Address,
    pub config: Address,
    pub match_id: Vec<u8>,
}

impl TestEnv {
    pub fn new() -> Self {
        let mut svm = LiteSVM::new();
        let program_id = blockchain::id();

        let blockchain_bytes = include_bytes!(concat!(
            env!("CARGO_TARGET_TMPDIR"),
            "/../deploy/blockchain.so"
        ));
        let txline_bytes = include_bytes!(concat!(
            env!("CARGO_TARGET_TMPDIR"),
            "/../deploy/txline_mock.so"
        ));

        svm.add_program(program_id, blockchain_bytes).unwrap();
        svm.add_program(TXLINE_MOCK_PROGRAM_ID, txline_bytes)
            .unwrap();

        let authority = Keypair::new();
        let maker = Keypair::new();
        let taker = Keypair::new();
        let mint = Keypair::new();

        for kp in [&authority, &maker, &taker] {
            svm.airdrop(&kp.pubkey(), 10_000_000_000).unwrap();
        }

        let mut env = Self {
            svm,
            authority,
            maker,
            taker,
            mint: mint.pubkey(),
            config: Address::default(),
            match_id: b"wc-match-001".to_vec(),
        };

        env.init_mint(&mint);
        env.init_config();
        env.register_wallet(&env.maker.pubkey(), [1u8; 32]);
        env.register_wallet(&env.taker.pubkey(), [2u8; 32]);
        env.fund_wallet(&env.maker.pubkey(), 5_000_000);
        env.fund_wallet(&env.taker.pubkey(), 5_000_000);
        env
    }

    fn send(&mut self, signers: &[Keypair], instructions: &[Instruction]) {
        let payer = signers[0].pubkey();
        let blockhash = self.svm.latest_blockhash();
        let signer_refs: Vec<&Keypair> = signers.iter().collect();
        let tx =
            Transaction::new_signed_with_payer(instructions, Some(&payer), &signer_refs, blockhash);
        let result = self.svm.send_transaction(tx);
        assert!(result.is_ok(), "tx failed: {:?}", result.err());
    }

    fn init_mint(&mut self, mint: &Keypair) {
        let mint_pk = mint.pubkey();
        let payer = self.authority.pubkey();
        let mint_len = spl_token_interface::state::Mint::LEN as u64;
        let rent = self
            .svm
            .minimum_balance_for_rent_exemption(mint_len as usize);

        let create_ix = create_account(&payer, &mint_pk, rent, mint_len, &TOKEN_PROGRAM_ID);
        let init_ix = initialize_mint2(&TOKEN_PROGRAM_ID, &mint_pk, &payer, None, 6).unwrap();

        self.send(
            &[self.authority.insecure_clone(), mint.insecure_clone()],
            &[create_ix, init_ix],
        );
    }

    fn create_ata(&mut self, owner: &Address) -> Address {
        let ata = get_associated_token_address(owner, &self.mint);
        let ix = create_associated_token_account(
            &self.authority.pubkey(),
            owner,
            &self.mint,
            &TOKEN_PROGRAM_ID,
        );
        self.send(&[self.authority.insecure_clone()], &[ix]);
        ata
    }

    pub fn fund_wallet(&mut self, owner: &Address, amount: u64) {
        let ata = self.create_ata(owner);
        let ix = mint_to(
            &TOKEN_PROGRAM_ID,
            &self.mint,
            &ata,
            &self.authority.pubkey(),
            &[],
            amount,
        )
        .unwrap();
        self.send(&[self.authority.insecure_clone()], &[ix]);
    }

    fn init_config(&mut self) {
        let (config, _) = Address::find_program_address(&[CONFIG_SEED], &blockchain::id());
        self.config = config;

        let ix = Instruction::new_with_bytes(
            blockchain::id(),
            &blockchain::instruction::Initialize {
                stablecoin_mint: self.mint,
                txline_program: TXLINE_MOCK_PROGRAM_ID,
            }
            .data(),
            blockchain::accounts::Initialize {
                authority: self.authority.pubkey(),
                config,
                system_program: solana_system_interface::program::ID,
            }
            .to_account_metas(None),
        );
        self.send(&[self.authority.insecure_clone()], &[ix]);

        let cfg = self.get::<Config>(&config);
        assert_eq!(cfg.authority, self.authority.pubkey());
        assert_eq!(cfg.stablecoin_mint, self.mint);
    }

    pub fn wallet_profile_pda(&self, wallet: &Address) -> Address {
        let (profile, _) = Address::find_program_address(
            &[WALLET_PROFILE_SEED, wallet.as_ref()],
            &blockchain::id(),
        );
        profile
    }

    pub fn register_wallet(&mut self, wallet: &Address, user_id_hash: [u8; 32]) {
        let profile = self.wallet_profile_pda(wallet);
        let ix = Instruction::new_with_bytes(
            blockchain::id(),
            &blockchain::instruction::RegisterWallet {
                wallet: *wallet,
                user_id_hash,
            }
            .data(),
            blockchain::accounts::RegisterWallet {
                authority: self.authority.pubkey(),
                config: self.config,
                wallet_profile: profile,
                system_program: solana_system_interface::program::ID,
            }
            .to_account_metas(None),
        );
        self.send(&[self.authority.insecure_clone()], &[ix]);
        let p: WalletProfile = self.get(&profile);
        assert_eq!(p.wallet, *wallet);
    }

    pub fn wager_pdas(&self, maker: &Address, match_id: &[u8]) -> (Address, Address) {
        let (wager, _) = Address::find_program_address(
            &[WAGER_SEED, maker.as_ref(), match_id],
            &blockchain::id(),
        );
        let (vault, _) =
            Address::find_program_address(&[VAULT_SEED, wager.as_ref()], &blockchain::id());
        (wager, vault)
    }

    pub fn get<T: AccountDeserialize>(&self, key: &Address) -> T {
        let account = self.svm.get_account(key).expect("missing account");
        let mut data: &[u8] = &account.data;
        T::try_deserialize(&mut data).expect("deserialize failed")
    }

    pub fn token_balance(&self, ata: &Address) -> u64 {
        let account = self.svm.get_account(ata).unwrap();
        TokenAccountState::unpack(&account.data).unwrap().amount
    }

    pub fn make_wager(&mut self, stake: u64, side: Side) -> (Address, Address) {
        self.make_wager_for(stake, side, Address::default())
    }

    pub fn make_wager_for(&mut self, stake: u64, side: Side, invited_taker: Address) -> (Address, Address) {
        let maker = self.maker.pubkey();
        let (wager, vault) = self.wager_pdas(&maker, &self.match_id);
        let maker_ata = get_associated_token_address(&maker, &self.mint);

        let ix = Instruction::new_with_bytes(
            blockchain::id(),
            &blockchain::instruction::MakeWager {
                match_id: self.match_id.clone(),
                stake_amount: stake,
                maker_side: side,
                invited_taker,
            }
            .data(),
            blockchain::accounts::MakeWager {
                maker,
                config: self.config,
                wager,
                vault,
                maker_stablecoin: maker_ata,
                stablecoin_mint: self.mint,
                maker_wallet_profile: self.wallet_profile_pda(&maker),
                token_program: TOKEN_PROGRAM_ID,
                associated_token_program: ATA_PROGRAM_ID,
                system_program: solana_system_interface::program::ID,
            }
            .to_account_metas(None),
        );
        self.send(&[self.maker.insecure_clone()], &[ix]);

        let w: Wager = self.get(&wager);
        assert_eq!(w.status, WagerStatus::Open);
        assert_eq!(w.maker_side, side);
        (wager, vault)
    }

    pub fn accept_wager(&mut self, wager: Address, vault: Address, taker_side: Side) {
        let maker = self.maker.pubkey();
        let taker = self.taker.pubkey();
        let taker_ata = get_associated_token_address(&taker, &self.mint);

        let ix = Instruction::new_with_bytes(
            blockchain::id(),
            &blockchain::instruction::AcceptWager { taker_side }.data(),
            blockchain::accounts::AcceptWager {
                taker,
                config: self.config,
                wager,
                maker,
                taker_stablecoin: taker_ata,
                vault,
                stablecoin_mint: self.mint,
                taker_wallet_profile: self.wallet_profile_pda(&taker),
                token_program: TOKEN_PROGRAM_ID,
                associated_token_program: ATA_PROGRAM_ID,
            }
            .to_account_metas(None),
        );
        self.send(&[self.taker.insecure_clone()], &[ix]);

        let w: Wager = self.get(&wager);
        assert_eq!(w.status, WagerStatus::Matched);
        assert_eq!(w.taker, taker);
        assert_eq!(w.taker_side, taker_side);
    }

    pub fn cancel_wager(&mut self, wager: Address, vault: Address) {
        let maker = self.maker.pubkey();
        let maker_ata = get_associated_token_address(&maker, &self.mint);

        let ix = Instruction::new_with_bytes(
            blockchain::id(),
            &blockchain::instruction::CancelWager {}.data(),
            blockchain::accounts::CancelWager {
                maker,
                config: self.config,
                wager,
                vault,
                maker_stablecoin: maker_ata,
                stablecoin_mint: self.mint,
                maker_wallet_profile: self.wallet_profile_pda(&maker),
                token_program: TOKEN_PROGRAM_ID,
                associated_token_program: ATA_PROGRAM_ID,
            }
            .to_account_metas(None),
        );
        self.send(&[self.maker.insecure_clone()], &[ix]);
    }

    pub fn setup_daily_scores_account(&mut self) -> Address {
        let pda = Address::new_unique();
        self.svm
            .set_account(
                pda,
                Account {
                    lamports: 1_000_000_000,
                    data: vec![0u8; 8],
                    owner: TXLINE_MOCK_PROGRAM_ID,
                    ..Default::default()
                },
            )
            .unwrap();
        pda
    }

    pub fn minimal_validation(&self) -> ValidateStatArgs {
        ValidateStatArgs {
            ts: 1_700_000_000_000,
            fixture_summary: blockchain::cpi::ScoresBatchSummary {
                fixture_id: 1,
                update_stats: blockchain::cpi::ScoresUpdateStats {
                    update_count: 1,
                    min_timestamp: 1_700_000_000_000,
                    max_timestamp: 1_700_000_000_000,
                },
                events_sub_tree_root: [1u8; 32],
            },
            fixture_proof: vec![],
            main_tree_proof: vec![],
            predicate: blockchain::cpi::TraderPredicate {
                threshold: 0,
                comparison: blockchain::cpi::Comparison::GreaterThan,
            },
            stat_a: blockchain::cpi::StatTerm {
                stat_to_prove: blockchain::cpi::ScoreStat {
                    key: 1002,
                    value: 1,
                    period: 0,
                },
                event_stat_root: [2u8; 32],
                stat_proof: vec![],
            },
            stat_b: None,
            op: None,
        }
    }

    pub fn settle_wager(
        &mut self,
        wager: Address,
        vault: Address,
        winning_side: Side,
        winner_kp: &Keypair,
    ) {
        let winner = winner_kp.pubkey();
        let winner_ata = get_associated_token_address(&winner, &self.mint);
        if self.svm.get_account(&winner_ata).is_none() {
            self.create_ata(&winner);
        }
        let daily_scores = self.setup_daily_scores_account();
        let merkle_root = [9u8; 32];

        let ix = Instruction::new_with_bytes(
            blockchain::id(),
            &blockchain::instruction::SettleWager {
                validation: self.minimal_validation(),
                winning_side,
                merkle_root,
            }
            .data(),
            blockchain::accounts::SettleWager {
                settler: winner,
                config: self.config,
                wager,
                vault,
                winner,
                winner_stablecoin: winner_ata,
                stablecoin_mint: self.mint,
                daily_scores_merkle_roots: daily_scores,
                txline_program: TXLINE_MOCK_PROGRAM_ID,
                token_program: TOKEN_PROGRAM_ID,
                associated_token_program: ATA_PROGRAM_ID,
            }
            .to_account_metas(None),
        );
        self.send(&[winner_kp.insecure_clone()], &[ix]);
    }

    pub fn settle_wager_as_keeper(
        &mut self,
        wager: Address,
        vault: Address,
        winning_side: Side,
        winner: Address,
    ) {
        let winner_ata = get_associated_token_address(&winner, &self.mint);
        if self.svm.get_account(&winner_ata).is_none() {
            self.create_ata(&winner);
        }
        let daily_scores = self.setup_daily_scores_account();
        let merkle_root = [9u8; 32];

        let ix = Instruction::new_with_bytes(
            blockchain::id(),
            &blockchain::instruction::SettleWager {
                validation: self.minimal_validation(),
                winning_side,
                merkle_root,
            }
            .data(),
            blockchain::accounts::SettleWager {
                settler: self.authority.pubkey(),
                config: self.config,
                wager,
                vault,
                winner,
                winner_stablecoin: winner_ata,
                stablecoin_mint: self.mint,
                daily_scores_merkle_roots: daily_scores,
                txline_program: TXLINE_MOCK_PROGRAM_ID,
                token_program: TOKEN_PROGRAM_ID,
                associated_token_program: ATA_PROGRAM_ID,
            }
            .to_account_metas(None),
        );
        self.send(&[self.authority.insecure_clone()], &[ix]);
    }
}
