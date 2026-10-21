import { InputHTMLAttributes, forwardRef } from 'react';
import { cn } from '../../lib/utils';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, id, ...props }, ref) => {
    return (
      <div className="w-full space-y-1.5">
        {label && (
          <label htmlFor={id} className="block text-xs font-mono text-brand-textMuted uppercase tracking-wider">
            {label}
          </label>
        )}
        <input
          id={id}
          ref={ref}
          className={cn(
            'w-full bg-brand-surface border border-brand-border rounded px-3 py-2 text-sm text-brand-textHeading placeholder:text-zinc-600 focus:outline-none focus:border-brand-amber/80 focus:ring-1 focus:ring-brand-amber/40 transition-colors',
            error && 'border-red-500/80 focus:border-red-500 focus:ring-red-500/20',
            className
          )}
          {...props}
        />
        {error && <p className="text-xs text-red-400 font-mono">{error}</p>}
      </div>
    );
  }
);

Input.displayName = 'Input';
