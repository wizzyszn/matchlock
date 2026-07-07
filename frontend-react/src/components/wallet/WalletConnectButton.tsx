import { useWallet } from '@solana/wallet-adapter-react'
import { WalletReadyState, type WalletName } from '@solana/wallet-adapter-base'
import { Loader2, LogOut, Wallet } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

function truncateAddress(address: string): string {
  return `${address.slice(0, 4)}…${address.slice(-4)}`
}

export function WalletConnectButton() {
  const { connected, connect, connecting, disconnect, publicKey, select, wallets } =
    useWallet()
  const [pendingWallet, setPendingWallet] = useState<string | null>(null)

  const installedWallets = useMemo(
    () =>
      wallets.filter(
        (wallet) => wallet.readyState === WalletReadyState.Installed,
      ),
    [wallets],
  )

  const handleConnect = useCallback(
    async (walletName: string) => {
      setPendingWallet(walletName)
      try {
        select(walletName as WalletName)
        await connect()
      } finally {
        setPendingWallet(null)
      }
    },
    [connect, select],
  )

  if (connected && publicKey) {
    return (
      <DropdownMenu>
        <DropdownMenuTrigger className="inline-flex">
          <Button variant="outline" className="gap-2 font-mono" type="button">
            <Wallet className="size-4" />
            {truncateAddress(publicKey.toBase58())}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          <DropdownMenuGroup>
            <DropdownMenuLabel
              className="truncate font-mono text-xs font-normal text-muted-foreground"
              title={publicKey.toBase58()}
            >
              {publicKey.toBase58()}
            </DropdownMenuLabel>
            <DropdownMenuItem
              variant="destructive"
              onClick={() => void disconnect()}
            >
              <LogOut className="size-4" />
              Disconnect
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    )
  }

  if (connecting || pendingWallet) {
    return (
      <Button disabled className="gap-2">
        <Loader2 className="size-4 animate-spin" />
        Connecting…
      </Button>
    )
  }

  if (installedWallets.length === 0) {
    return (
      <Button variant="outline" disabled className="gap-2">
        <Wallet className="size-4" />
        No wallet found
      </Button>
    )
  }

  if (installedWallets.length === 1) {
    const wallet = installedWallets[0]
    return (
      <Button
        className="gap-2"
        onClick={() => void handleConnect(wallet.adapter.name)}
      >
        <Wallet className="size-4" />
        Connect {wallet.adapter.name}
      </Button>
    )
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="inline-flex">
        <Button className="gap-2" type="button">
          <Wallet className="size-4" />
          Connect wallet
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52">
        <DropdownMenuGroup>
          <DropdownMenuLabel>Select wallet</DropdownMenuLabel>
          {installedWallets.map((wallet) => (
            <DropdownMenuItem
              key={wallet.adapter.name}
              onClick={() => void handleConnect(wallet.adapter.name)}
            >
              {wallet.adapter.name}
            </DropdownMenuItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}