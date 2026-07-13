import { forwardRef, type HTMLAttributes } from "react"

import { cn } from "@/lib/utils"

const PageHeader = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("mb-8 space-y-1", className)} {...props} />
  )
)
PageHeader.displayName = "PageHeader"

const PageHeaderHeading = forwardRef<HTMLHeadingElement, HTMLAttributes<HTMLHeadingElement>>(
  ({ className, ...props }, ref) => {
    return (
      <h1
        ref={ref}
        className={cn(
          "font-heading text-3xl font-medium leading-tight tracking-tight sm:text-4xl",
          className
        )}
        {...props}
      />
    )
  }
)
PageHeaderHeading.displayName = "PageHeaderHeading"

const PageHeaderDescription = forwardRef<HTMLParagraphElement, HTMLAttributes<HTMLParagraphElement>>(
  ({ className, ...props }, ref) => (
    <p
      ref={ref}
      className={cn("max-w-prose text-sm text-muted-foreground", className)}
      {...props}
    />
  )
)
PageHeaderDescription.displayName = "PageHeaderDescription"

export { PageHeader, PageHeaderHeading, PageHeaderDescription }
