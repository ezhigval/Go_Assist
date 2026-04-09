/**
 * Button component - versatile button with multiple variants and states
 * Follows shadcn/ui patterns with Modulr styling
 */

import React, { forwardRef } from 'react';
import { cn } from '@modulr/lib/utils';
import type { ButtonHTMLAttributes, ReactNode } from 'react';

// ============================================================================
// BUTTON VARIANTS
// ============================================================================

export interface ButtonVariant {
  base: string;
  hover: string;
  active: string;
  disabled: string;
  focus: string;
}

export const buttonVariants: Record<string, ButtonVariant> = {
  primary: {
    base: 'bg-primary-600 text-white border-primary-600 shadow-sm',
    hover: 'bg-primary-700 border-primary-700 shadow-md',
    active: 'bg-primary-800 border-primary-800 shadow-lg',
    disabled: 'bg-primary-300 border-primary-300 text-primary-100 cursor-not-allowed',
    focus: 'ring-2 ring-primary-500 ring-offset-2',
  },
  secondary: {
    base: 'bg-transparent text-gray-700 border-gray-300 shadow-sm',
    hover: 'bg-gray-50 border-gray-400 shadow-md',
    active: 'bg-gray-100 border-gray-500 shadow-lg',
    disabled: 'bg-gray-100 border-gray-200 text-gray-400 cursor-not-allowed',
    focus: 'ring-2 ring-gray-500 ring-offset-2',
  },
  outline: {
    base: 'bg-transparent text-primary-600 border-primary-600',
    hover: 'bg-primary-50 border-primary-700',
    active: 'bg-primary-100 border-primary-800',
    disabled: 'bg-transparent text-primary-300 border-primary-300 cursor-not-allowed',
    focus: 'ring-2 ring-primary-500 ring-offset-2',
  },
  ghost: {
    base: 'bg-transparent text-gray-700 border-transparent',
    hover: 'bg-gray-100',
    active: 'bg-gray-200',
    disabled: 'bg-transparent text-gray-400 cursor-not-allowed',
    focus: 'ring-2 ring-gray-500 ring-offset-2',
  },
  destructive: {
    base: 'bg-red-600 text-white border-red-600 shadow-sm',
    hover: 'bg-red-700 border-red-700 shadow-md',
    active: 'bg-red-800 border-red-800 shadow-lg',
    disabled: 'bg-red-300 border-red-300 text-red-100 cursor-not-allowed',
    focus: 'ring-2 ring-red-500 ring-offset-2',
  },
  success: {
    base: 'bg-green-600 text-white border-green-600 shadow-sm',
    hover: 'bg-green-700 border-green-700 shadow-md',
    active: 'bg-green-800 border-green-800 shadow-lg',
    disabled: 'bg-green-300 border-green-300 text-green-100 cursor-not-allowed',
    focus: 'ring-2 ring-green-500 ring-offset-2',
  },
  scope: {
    base: 'bg-transparent border-2 shadow-sm',
    hover: 'shadow-md transform scale-105',
    active: 'shadow-lg transform scale-105',
    disabled: 'opacity-50 cursor-not-allowed',
    focus: 'ring-2 ring-offset-2',
  },
};

export const buttonSizes: Record<string, string> = {
  xs: 'px-2 py-1 text-xs rounded',
  sm: 'px-3 py-1.5 text-sm rounded-md',
  md: 'px-4 py-2 text-sm rounded-md',
  lg: 'px-6 py-3 text-base rounded-lg',
  xl: 'px-8 py-4 text-lg rounded-lg',
  icon: 'p-2 rounded-md',
};

// ============================================================================
// BUTTON PROPS
// ============================================================================

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

// ============================================================================
// BUTTON COMPONENT
// ============================================================================

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
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
  ) => {
    const variantStyles = buttonVariants[variant] || buttonVariants.primary;
    const sizeStyles = buttonSizes[size] || buttonSizes.md;

    // Apply scope color for scope variant
    const scopeStyles = variant === 'scope' && scopeColor
      ? {
          borderColor: scopeColor,
          color: scopeColor,
          '--tw-ring-color': scopeColor,
        } as React.CSSProperties
      : {};

    const buttonClasses = cn(
      // Base styles
      'inline-flex items-center justify-center gap-2 font-medium transition-all duration-200 ease-in-out',
      'focus:outline-none focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed',
      'border rounded-md',
      
      // Variant styles
      variantStyles.base,
      variantStyles.hover,
      variantStyles.active,
      variantStyles.focus,
      
      // Size styles
      sizeStyles,
      
      // State styles
      disabled && variantStyles.disabled,
      loading && 'cursor-wait',
      fullWidth && 'w-full',
      
      // Custom className
      className
    );

    return (
      <button
        className={buttonClasses}
        ref={ref}
        disabled={disabled || loading}
        style={scopeStyles}
        {...props}
      >
        {/* Loading spinner */}
        {loading && (
          <svg
            className="animate-spin -ml-1 mr-2 h-4 w-4"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
        )}

        {/* Left icon */}
        {leftIcon && !loading && (
          <span className="flex-shrink-0">{leftIcon}</span>
        )}

        {/* Button content */}
        <span className={loading ? 'opacity-70' : ''}>
          {children}
        </span>

        {/* Right icon */}
        {rightIcon && !loading && (
          <span className="flex-shrink-0">{rightIcon}</span>
        )}
      </button>
    );
  }
);

Button.displayName = 'Button';

// ============================================================================
// SPECIALIZED BUTTON COMPONENTS
// ============================================================================

/**
 * Scope button - for scope selection with color indication
 */
export interface ScopeButtonProps extends Omit<ButtonProps, 'variant'> {
  segment: string;
  isActive?: boolean;
}

export function ScopeButton({
  segment,
  isActive = false,
  className,
  children,
  ...props
}: ScopeButtonProps) {
  const scopeColors: Record<string, string> = {
    personal: '#10b981',
    work: '#3b82f6',
    family: '#f59e0b',
    health: '#ef4444',
    learning: '#8b5cf6',
    finance: '#06b6d4',
    creative: '#ec4899',
    social: '#84cc16',
  };

  const color = scopeColors[segment] || '#6b7280';

  return (
    <Button
      variant="scope"
      scopeColor={color}
      className={cn(
        'font-semibold',
        isActive && 'ring-2 ring-offset-2',
        className
      )}
      style={{
        borderColor: isActive ? color : undefined,
        color: isActive ? color : undefined,
        backgroundColor: isActive ? `${color}10` : undefined,
        '--tw-ring-color': color,
      } as React.CSSProperties}
      {...props}
    >
      {children}
    </Button>
  );
}

/**
 * Icon button - button with only an icon
 */
export interface IconButtonProps extends Omit<ButtonProps, 'children' | 'size'> {
  icon: ReactNode;
  tooltip?: string;
  size?: 'xs' | 'sm' | 'md' | 'lg';
}

export function IconButton({
  icon,
  tooltip,
  size = 'md',
  className,
  ...props
}: IconButtonProps) {
  const sizeMap = {
    xs: 'p-1 text-xs',
    sm: 'p-1.5 text-sm',
    md: 'p-2 text-base',
    lg: 'p-3 text-lg',
  };

  return (
    <Button
      variant="ghost"
      size="icon"
      className={cn(sizeMap[size], className)}
      title={tooltip}
      {...props}
    >
      {icon}
    </Button>
  );
}

/**
 * Floating action button
 */
export interface FabProps extends Omit<ButtonProps, 'variant' | 'size'> {
  position?: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left';
}

export function Fab({
  position = 'bottom-right',
  className,
  children,
  ...props
}: FabProps) {
  const positionStyles: Record<string, string> = {
    'bottom-right': 'bottom-6 right-6',
    'bottom-left': 'bottom-6 left-6',
    'top-right': 'top-6 right-6',
    'top-left': 'top-6 left-6',
  };

  return (
    <Button
      variant="primary"
      size="lg"
      className={cn(
        'fixed rounded-full shadow-lg z-50',
        positionStyles[position],
        className
      )}
      {...props}
    >
      {children}
    </Button>
  );
}

// ============================================================================
// BUTTON GROUP
// ============================================================================

export interface ButtonGroupProps {
  children: ReactNode;
  className?: string;
  vertical?: boolean;
}

export function ButtonGroup({
  children,
  className,
  vertical = false,
}: ButtonGroupProps) {
  return (
    <div
      className={cn(
        'inline-flex',
        vertical ? 'flex-col' : 'flex-row',
        className
      )}
    >
      {React.Children.map(children, (child, index) => {
        if (React.isValidElement(child)) {
          return React.cloneElement(child, {
            className: cn(
              index > 0 && !vertical && '-ml-px',
              index > 0 && vertical && '-mt-px',
              child.props.className
            ),
          });
        }
        return child;
      })}
    </div>
  );
}

// ============================================================================
// EXPORTS
// ============================================================================

export {
  Button,
  ScopeButton,
  IconButton,
  Fab,
  ButtonGroup,
};

export type {
  ButtonProps,
  ScopeButtonProps,
  IconButtonProps,
  FabProps,
  ButtonGroupProps,
};
