import { forwardRef, type ButtonHTMLAttributes, type CSSProperties, type ReactNode } from 'react';
import { cn } from '@modulr/lib/utils';

const buttonVariants = {
  primary: 'bg-primary-600 text-white border-primary-600 hover:bg-primary-700',
  secondary: 'bg-white text-gray-800 border-gray-300 hover:bg-gray-50',
  outline: 'bg-transparent text-primary-700 border-primary-500 hover:bg-primary-50',
  ghost: 'bg-transparent text-gray-700 border-transparent hover:bg-gray-100',
  destructive: 'bg-red-600 text-white border-red-600 hover:bg-red-700',
  success: 'bg-green-600 text-white border-green-600 hover:bg-green-700',
  scope: 'bg-transparent border-2',
} as const;

const buttonSizes = {
  xs: 'px-2 py-1 text-xs rounded',
  sm: 'px-3 py-1.5 text-sm rounded-md',
  md: 'px-4 py-2 text-sm rounded-md',
  lg: 'px-6 py-3 text-base rounded-lg',
  xl: 'px-8 py-4 text-lg rounded-lg',
  icon: 'p-2 rounded-md',
} as const;

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: keyof typeof buttonVariants;
  size?: keyof typeof buttonSizes;
  loading?: boolean;
  leftIcon?: ReactNode;
  rightIcon?: ReactNode;
  fullWidth?: boolean;
  scopeColor?: string;
  children: ReactNode;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    className,
    variant = 'primary',
    size = 'md',
    loading = false,
    leftIcon,
    rightIcon,
    fullWidth = false,
    scopeColor,
    disabled,
    children,
    ...props
  },
  ref
) {
  const scopeStyles: CSSProperties | undefined =
    variant === 'scope' && scopeColor
      ? {
          borderColor: scopeColor,
          color: scopeColor,
        }
      : undefined;

  return (
    <button
      ref={ref}
      disabled={disabled || loading}
      style={scopeStyles}
      className={cn(
        'inline-flex items-center justify-center gap-2 border font-medium transition-colors',
        'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        buttonVariants[variant],
        buttonSizes[size],
        fullWidth && 'w-full',
        className
      )}
      {...props}
    >
      {loading && <span className="inline-block h-3 w-3 animate-pulse rounded-full bg-current opacity-60" />}
      {!loading && leftIcon}
      <span>{children}</span>
      {!loading && rightIcon}
    </button>
  );
});

export interface ScopeButtonProps extends Omit<ButtonProps, 'variant'> {
  segment: string;
  isActive?: boolean;
}

const scopeColors: Record<string, string> = {
  personal: '#10b981',
  family: '#f59e0b',
  work: '#3b82f6',
  business: '#0ea5e9',
  health: '#ef4444',
  travel: '#8b5cf6',
  pets: '#f97316',
  assets: '#64748b',
};

export function ScopeButton({ segment, isActive = false, className, children, ...props }: ScopeButtonProps) {
  const color = scopeColors[segment] ?? '#6b7280';
  return (
    <Button
      variant="scope"
      scopeColor={color}
      className={cn(isActive && 'ring-2 ring-offset-2', className)}
      style={{
        borderColor: color,
        color,
        backgroundColor: isActive ? `${color}15` : undefined,
      }}
      {...props}
    >
      {children}
    </Button>
  );
}

export interface IconButtonProps extends Omit<ButtonProps, 'children'> {
  icon: ReactNode;
  label: string;
}

export function IconButton({ icon, label, ...props }: IconButtonProps) {
  return (
    <Button {...props}>
      <span className="sr-only">{label}</span>
      {icon}
    </Button>
  );
}

export interface FabProps extends Omit<ButtonProps, 'size'> {
  bottom?: number;
  right?: number;
}

export function Fab({ bottom = 24, right = 24, className, ...props }: FabProps) {
  return (
    <Button
      {...props}
      size="lg"
      className={cn('fixed z-50 rounded-full shadow-lg', className)}
      style={{ bottom, right, ...(props.style ?? {}) }}
    />
  );
}

export interface ButtonGroupProps {
  children: ReactNode;
  className?: string;
}

export function ButtonGroup({ children, className }: ButtonGroupProps) {
  return <div className={cn('inline-flex items-center gap-2', className)}>{children}</div>;
}
