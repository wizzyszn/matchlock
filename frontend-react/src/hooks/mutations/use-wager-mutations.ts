import { useAnchorWallet, useConnection } from "@solana/wallet-adapter-react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { PublicKey } from "@solana/web3.js";

import { useConfig } from "@/hooks/use-api";
import type { Side, Wager, WagerStatus, WagerSettlementStatus } from "@/lib/api";
import { mapTransactionError } from "@/lib/errors";
import { getProgram, getUsdtMint } from "@/lib/anchor";
import { queryKeys } from "@/lib/query-keys";

import { useApi } from "@/hooks/use-api";
import { useWalletLinkStatus } from "@/hooks/use-wallet-link-status";
import { useOptimisticWagersStore } from "@/stores/optimistic-wagers-store";
import { classifyMatch } from "@/lib/match-display";
import {
  buildAcceptWagerTransaction,
  buildCancelWagerTransaction,
  buildClaimWagerTransaction,
  buildMakeWagerTransaction,
  sendTransaction,
  simulateTransaction,
} from "@/lib/wager-tx";

export type TxAction = "make" | "accept" | "cancel" | "claim";

async function syncLeaderboardClaim(
  sync: (wagerPubkey: string, txSignature?: string) => Promise<{ synced: boolean }>,
  wagerPubkey: string,
  txSignature: string,
) {
  for (let attempt = 0; attempt < 5; attempt += 1) {
    try {
      await sync(wagerPubkey, txSignature);
      return;
    } catch {
      if (attempt === 4) {
        return;
      }
      await new Promise((resolve) => window.setTimeout(resolve, 1_500 * (attempt + 1)));
    }
  }
}

export function useWagerMutations() {
  const { connection } = useConnection();
  const wallet = useAnchorWallet();
  const { canTransact, connected, needsLink, conflict } = useWalletLinkStatus();
  const config = useConfig();
  const api = useApi();
  const queryClient = useQueryClient();
  const optimisticWagers = useOptimisticWagersStore();

  const assertMatchOpenForWagering = async (matchId: string) => {
    const match = await api.getMatch(matchId);
    const phase = classifyMatch(match);
    if (phase === "finished") {
      throw new Error("This fixture is no longer available for wagering.");
    }
    return match;
  };

  const invalidateWagers = async () => {
    await queryClient.invalidateQueries({ queryKey: ["wagers"] });
    await queryClient.invalidateQueries({ queryKey: ["tokenBalance"] });
  };

  const getCachedWager = (wagerPubkey: string) => {
    const detail = queryClient.getQueryData<Wager>(
      queryKeys.wagers.detail(wagerPubkey),
    );
    if (detail) {
      return detail;
    }

    const listEntries = queryClient.getQueriesData<Wager[]>({ queryKey: ["wagers"] });
    for (const [, wagers] of listEntries) {
      const found = wagers?.find((wager) => wager.pubkey === wagerPubkey);
      if (found) {
        return found;
      }
    }

    return null;
  };

  const updateWagerStatus = (wagerPubkey: string, status: WagerStatus) => {
    queryClient.setQueriesData<Wager[]>({ queryKey: ["wagers"] }, (old) => {
      if (!Array.isArray(old)) return old;
      return old.map((w) => (w.pubkey === wagerPubkey ? { ...w, status } : w));
    });
    queryClient.setQueryData<Wager>(queryKeys.wagers.detail(wagerPubkey), (old) => {
      if (!old) return old;
      return { ...old, status };
    });
  };

  const makeWager = useMutation({
    mutationFn: async (input: {
      matchId: string;
      stake: bigint;
      makerSide: Side;
      participant1IsHome: boolean;
      invitedTaker?: PublicKey;
    }) => {
      if (!wallet?.publicKey) {
        throw new Error("Connect your wallet on Profile first.");
      }
      if (conflict) {
        throw new Error(
          "This wallet is linked to another Matchlock account. Switch wallet on Profile.",
        );
      }
      if (needsLink) {
        throw new Error(
          "Link your connected wallet to your account on Profile.",
        );
      }
      const program = getProgram(connection, wallet);
      const stablecoinMint = getUsdtMint(config);
      const match = await assertMatchOpenForWagering(input.matchId);

      const { tx, wagerPubkey } = await buildMakeWagerTransaction({
        program,
        connection,
        wallet,
        matchId: input.matchId,
        stake: input.stake,
        makerSide: input.makerSide,
        participant1IsHome: match.participant1_is_home,
        stablecoinMint,
        invitedTaker: input.invitedTaker,
      });

      await simulateTransaction(connection, wallet, tx);
      const signature = await sendTransaction(connection, wallet, tx);
      return {
        signature,
        wagerPubkey: wagerPubkey.toBase58(),
        stake: Number(input.stake),
        matchId: input.matchId,
        makerSide: input.makerSide,
        invitedTaker: input.invitedTaker?.toBase58(),
        maker: wallet.publicKey.toBase58(),
      };
    },
    onSuccess: (data) => {
      optimisticWagers.upsert({
        pubkey: data.wagerPubkey,
        maker: data.maker,
        invited_taker: data.invitedTaker,
        taker: "11111111111111111111111111111111",
        match_id: data.matchId,
        maker_side: data.makerSide,
        stake: data.stake,
        status: "open",
      });
      void invalidateWagers();
    },
  });

  const acceptWager = useMutation({
    mutationFn: async (input: {
      wagerPubkey: string;
      maker: string;
      takerSide: Side;
      matchId: string;
    }) => {
      if (!wallet?.publicKey) {
        throw new Error("Connect your wallet on Profile first.");
      }
      if (conflict) {
        throw new Error(
          "This wallet is linked to another Matchlock account. Switch wallet on Profile.",
        );
      }
      if (needsLink) {
        throw new Error(
          "Link your connected wallet to your account on Profile.",
        );
      }
      const program = getProgram(connection, wallet);
      const stablecoinMint = getUsdtMint(config);
      await assertMatchOpenForWagering(input.matchId);

      const tx = await buildAcceptWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        maker: new PublicKey(input.maker),
        matchId: input.matchId,
        takerSide: input.takerSide,
        stablecoinMint,
      });

      await simulateTransaction(connection, wallet, tx);
      return sendTransaction(connection, wallet, tx);
    },
    onSuccess: (_data, input) => {
      const cachedWager = getCachedWager(input.wagerPubkey);
      if (cachedWager) {
        optimisticWagers.markAccepted(cachedWager);
        updateWagerStatus(input.wagerPubkey, "matched");
      }
      invalidateWagers();
    },
  });

  const cancelWager = useMutation({
    mutationFn: async (input: { wagerPubkey: string; wager?: Wager }) => {
      if (!wallet?.publicKey) {
        throw new Error("Connect your wallet on Profile first.");
      }
      if (conflict) {
        throw new Error(
          "This wallet is linked to another Matchlock account. Switch wallet on Profile.",
        );
      }
      if (needsLink) {
        throw new Error(
          "Link your connected wallet to your account on Profile.",
        );
      }
      const program = getProgram(connection, wallet);
      const stablecoinMint = getUsdtMint(config);

      const tx = await buildCancelWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        stablecoinMint,
      });

      await simulateTransaction(connection, wallet, tx);
      const signature = await sendTransaction(connection, wallet, tx);
      return { signature, wagerPubkey: input.wagerPubkey };
    },
    onMutate: async (input) => {
      await queryClient.cancelQueries({ queryKey: ["wagers"] });
      const cachedWager = input.wager ?? getCachedWager(input.wagerPubkey);
      if (cachedWager) {
        optimisticWagers.markCancelled(cachedWager);
        updateWagerStatus(input.wagerPubkey, "cancelled");
      }
      return { previousWager: cachedWager };
    },
    onSuccess: (data, input) => {
      updateWagerStatus(data.wagerPubkey, "cancelled");
      const cachedWager = input.wager ?? getCachedWager(data.wagerPubkey);
      if (cachedWager) {
        optimisticWagers.markCancelled(cachedWager);
      }
      invalidateWagers();
    },
    onError: (_error, input, context) => {
      const previousWager = context?.previousWager ?? input.wager;
      if (previousWager) {
        optimisticWagers.upsert(previousWager);
        queryClient.setQueryData(
          queryKeys.wagers.detail(previousWager.pubkey),
          previousWager,
        );
        queryClient.setQueriesData<Wager[]>({ queryKey: ["wagers"] }, (old) => {
          if (!Array.isArray(old)) return old;
          return old.map((wager) =>
            wager.pubkey === previousWager.pubkey ? previousWager : wager,
          );
        });
      }
      void invalidateWagers();
    },
  });

  const claimWager = useMutation({
    mutationFn: async (input: { wagerPubkey: string }) => {
      if (!wallet?.publicKey) {
        throw new Error("Connect your wallet on Profile first.");
      }
      if (conflict) {
        throw new Error(
          "This wallet is linked to another Matchlock account. Switch wallet on Profile.",
        );
      }
      if (needsLink) {
        throw new Error(
          "Link your connected wallet to your account on Profile.",
        );
      }
      const proof = await api.getWagerSettlementProof(input.wagerPubkey);
      const program = getProgram(connection, wallet);
      const stablecoinMint = getUsdtMint(config);

      const tx = await buildClaimWagerTransaction({
        program,
        wallet,
        wagerPubkey: new PublicKey(input.wagerPubkey),
        proof,
        stablecoinMint,
      });

      await simulateTransaction(connection, wallet, tx);
      const signature = await sendTransaction(connection, wallet, tx);
      return { signature, wagerPubkey: input.wagerPubkey };
    },
    onSuccess: (data) => {
      const cachedWager = getCachedWager(data.wagerPubkey);
      if (cachedWager) {
        optimisticWagers.markClaimed(cachedWager);
      }
      updateWagerStatus(data.wagerPubkey, "settled");
      queryClient.setQueryData<WagerSettlementStatus>(
        queryKeys.wagers.settlement(data.wagerPubkey),
        (old) => {
          if (!old) return old;
          return { ...old, state: "settled", message: "Winnings have been sent" };
        },
      );
      void syncLeaderboardClaim(
        (wagerPubkey, txSignature) => api.syncLeaderboardSettlement(wagerPubkey, txSignature),
        data.wagerPubkey,
        data.signature,
      ).finally(() => {
        void queryClient.invalidateQueries({ queryKey: ["leaderboard"] });
      });
      invalidateWagers();
    },
  });

  return {
    makeWager,
    acceptWager,
    cancelWager,
    claimWager,
    mapError: mapTransactionError,
    isWalletReady: canTransact,
    walletConnected: connected,
    walletNeedsLink: needsLink,
    walletConflict: conflict,
  };
}
