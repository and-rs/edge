import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

// TODO: Add utilities for handling data-heavy operations
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

