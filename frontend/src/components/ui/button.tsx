import type { ButtonRootProps } from "@kobalte/core/button"
import { Button as ButtonPrimitive } from "@kobalte/core/button"
import type { PolymorphicProps } from "@kobalte/core/polymorphic"
import { cva, type VariantProps } from "class-variance-authority"
import { splitProps, type ValidComponent } from "solid-js"
import { cn } from "~/lib/utils"

export const buttonVariants = cva(
  "inline-flex items-center justify-center text-sm font-medium transition-all bg-inherit",
  {
    variants: {
      variant: {
        default: "bg-muted shadow-sm hover:bg-secondary",
        ghost: "hover:bg-muted",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 px-3 text-xs",
        lg: "h-10 px-8",
        icon: "h-8 w-8",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
)

type buttonProps<T extends ValidComponent = "button"> = ButtonRootProps<T> &
  VariantProps<typeof buttonVariants> & {
    class?: string
  }

export const Button = <T extends ValidComponent = "button">(
  props: PolymorphicProps<T, buttonProps<T>>,
) => {
  const [local, rest] = splitProps(props as buttonProps, [
    "class",
    "variant",
    "size",
  ])

  return (
    <ButtonPrimitive
      class={cn(
        buttonVariants({
          size: local.size,
          variant: local.variant,
        }),
        local.class,
      )}
      {...rest}
    />
  )
}
