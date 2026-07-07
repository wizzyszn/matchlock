import { useConnection, useWallet } from '@solana/wallet-adapter-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Buffer } from 'buffer'

import { useApi } from '@/hooks/use-api'
import { queryKeys } from '@/lib/query-keys'

export function useAuthMutations() {
  const api = useApi()
  const queryClient = useQueryClient()
  const { publicKey, signMessage } = useWallet()
  const { connection } = useConnection()

  const requestMagicLink = useMutation({
    mutationFn: (email: string) => api.requestMagicLink(email),
  })

  const verifyMagicLink = useMutation({
    mutationFn: (token: string) => api.verifyMagicLink(token),
    onSuccess: (profile) => {
      queryClient.setQueryData(queryKeys.auth.session, profile)
    },
  })

  const updateProfile = useMutation({
    mutationFn: (input: { display_name: string }) => api.updateProfile(input),
    onSuccess: (profile) => {
      queryClient.setQueryData(queryKeys.auth.session, profile)
    },
  })

  const logout = useMutation({
    mutationFn: () => api.logout(),
    onSuccess: () => {
      queryClient.setQueryData(queryKeys.auth.session, null)
      window.location.assign('/login')
    },
  })

  const linkWallet = useMutation({
    mutationFn: async (label?: string) => {
      if (!publicKey || !signMessage) {
        throw new Error('Connect your wallet in your browser first.')
      }
      const pubkey = publicKey.toBase58()
      const { message } = await api.getWalletLinkChallenge(pubkey)
      const encoded = new TextEncoder().encode(message)
      const signature = await signMessage(encoded)
      return api.linkWallet({
        pubkey,
        message,
        signature: Buffer.from(signature).toString('base64'),
        label,
      })
    },
    onSuccess: async () => {
      const profile = await api.getMe()
      queryClient.setQueryData(queryKeys.auth.session, profile)
      if (publicKey) {
        await queryClient.invalidateQueries({
          queryKey: queryKeys.auth.walletBinding(publicKey.toBase58()),
        })
      }
    },
  })

  const unlinkWallet = useMutation({
    mutationFn: (pubkey: string) => api.unlinkWallet(pubkey),
    onSuccess: async (_data, pubkey) => {
      const profile = await api.getMe()
      queryClient.setQueryData(queryKeys.auth.session, profile)
      await queryClient.invalidateQueries({
        queryKey: queryKeys.auth.walletBinding(pubkey),
      })
    },
  })

  const setPrimaryWallet = useMutation({
    mutationFn: (pubkey: string) => api.setPrimaryWallet(pubkey),
    onSuccess: async () => {
      const profile = await api.getMe()
      queryClient.setQueryData(queryKeys.auth.session, profile)
    },
  })

  const lookupUser = useMutation({
    mutationFn: (email: string) => api.lookupUser(email),
  })

  const createInvite = useMutation({
    mutationFn: (input: Parameters<typeof api.createInvite>[0]) =>
      api.createInvite(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.auth.invites })
    },
  })

  return {
    requestMagicLink,
    verifyMagicLink,
    updateProfile,
    logout,
    linkWallet,
    unlinkWallet,
    setPrimaryWallet,
    lookupUser,
    createInvite,
    connection,
    publicKey,
  }
}
