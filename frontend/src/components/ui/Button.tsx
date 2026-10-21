import React, { ButtonHTMLAttributes } from 'react';
import { cn } from '../../lib/utils';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
  isLoading?: boolean;
}

export const Button: React.FC<ButtonProps> = ({
  children,
  className,
  variant = 'primary',
  size = 'md',
  isLoading = false,
  disabled,
  ...props
}) => {
  const base =
    'inline-flex items-center justify-center font-medium transition-colors select-none focus:outline-none focus:ring-1 focus:ring-brand-amber disabled:opacity-50 disabled:pointer-events-none rounded';

  const variants = {
    primary: 'bg-brand-amber text-brand-black hover:bg-amber-400 font-semibold',
    secondary: 'bg-brand-card text-brand-textHeading hover:bg-zinc-800 border border-brand-border',
    danger: 'bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20',
    ghost: 'text-brand-textBody hover:text-brand-textHeading hover:bg-brand-card',
  };

  const sizes = {
    sm: 'text-xs px-2.5 py-1 gap-1.5 h-7',
    md: 'text-sm px-3.5 py-1.5 gap-2 h-9',
    lg: 'text-base px-4 py-2 gap-2.5 h-11',
  };

  return (
    <button
      className={cn(base, variants[variant], sizes[size], className)}
      disabled={disabled || isLoading}
      {...props}
    >
      {isLoading ? (
        <span className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-1" />
      ) : null}
      {children}
    </button>
  );
};
