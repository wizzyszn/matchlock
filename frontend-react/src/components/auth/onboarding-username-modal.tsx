import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Loader2, Sparkles } from 'lucide-react'

import { Dialog } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuthMutations } from '@/hooks/mutations/use-auth-mutations'
import { useSessionQuery } from '@/hooks/queries/use-session'
import { needsUsername } from '@/lib/display-name'

const formSchema = z.object({
  username: z
    .string()
    .regex(
      /^[a-zA-Z0-9_]{3,32}$/,
      '3–32 characters: letters, numbers, underscore',
    ),
})

type FormValues = z.infer<typeof formSchema>

export function OnboardingUsernameModal() {
  const { data: session } = useSessionQuery()
  const { updateProfile } = useAuthMutations()

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
  })

  const open = Boolean(session && needsUsername(session))

  if (!open) return null

  const onSubmit = async (data: FormValues) => {
    await updateProfile.mutateAsync({ display_name: data.username })
  }

  return (
    <Dialog
      open={open}
      onOpenChange={() => {}}
      dismissible={false}
      title="Choose your username"
      description="This is how other players see you in challenges and wagers."
      className="sm:max-w-md"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div className="flex items-center gap-3 rounded-lg border bg-muted/40 px-4 py-3">
          <Sparkles className="size-5 shrink-0 text-primary" aria-hidden />
          <p className="text-sm text-muted-foreground">
            Signed in as{' '}
            <span className="font-medium text-foreground">{session?.email}</span>
          </p>
        </div>

        <div className="space-y-2">
          <label htmlFor="onboarding-username" className="text-sm font-medium">
            Username
          </label>
          <Input
            id="onboarding-username"
            {...register('username')}
            placeholder="e.g. matchlock_ace"
            autoComplete="username"
            autoFocus
          />
          {errors.username ? (
            <p className="text-xs text-destructive" role="alert">
              {errors.username.message}
            </p>
          ) : (
            <p className="text-xs text-muted-foreground">
              3–32 characters: letters, numbers, underscore
            </p>
          )}
        </div>

        {updateProfile.error ? (
          <p className="text-sm text-destructive" role="alert">
            {updateProfile.error instanceof Error
              ? updateProfile.error.message
              : 'Could not save username'}
          </p>
        ) : null}

        <Button
          type="submit"
          className="w-full min-h-11"
          disabled={updateProfile.isPending}
        >
          {updateProfile.isPending ? (
            <>
              <Loader2 className="mr-2 size-4 animate-spin" aria-hidden />
              Saving…
            </>
          ) : (
            'Continue to Matchlock'
          )}
        </Button>
      </form>
    </Dialog>
  )
}
