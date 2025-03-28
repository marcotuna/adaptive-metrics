import React, { ButtonHTMLAttributes, ReactNode } from 'react';

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'link' | 'text';
export type ButtonSize = 'small' | 'medium' | 'large';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  /**
   * Button content
   */
  children: ReactNode;
  
  /**
   * Button style variant
   */
  variant?: ButtonVariant;
  
  /**
   * Button size
   */
  size?: ButtonSize;
  
  /**
   * Loading state
   */
  isLoading?: boolean;
  
  /**
   * Loading text to display when isLoading is true
   */
  loadingText?: string;
  
  /**
   * Icon to display before the button text
   */
  icon?: ReactNode;
  
  /**
   * Full width button
   */
  fullWidth?: boolean;
}

const Button: React.FC<ButtonProps> = ({
  children,
  variant = 'primary',
  size = 'medium',
  isLoading = false,
  loadingText,
  icon,
  fullWidth = false,
  disabled,
  className = '',
  ...rest
}) => {
  const baseClasses = "inline-flex items-center justify-center font-medium focus:outline-none focus:ring-2 focus:ring-offset-2 transition-colors rounded-md";
  
  const variantClasses = {
    primary: "bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500 disabled:bg-blue-300",
    secondary: "bg-gray-600 text-white hover:bg-gray-700 focus:ring-gray-500 disabled:bg-gray-300",
    danger: "bg-red-600 text-white hover:bg-red-700 focus:ring-red-500 disabled:bg-red-300",
    link: "text-indigo-600 hover:text-indigo-900 focus:ring-indigo-500 disabled:text-indigo-300",
    text: "text-red-600 hover:text-red-900 focus:ring-red-500 disabled:text-red-300"
  };
  
  const sizeClasses = {
    small: "px-3 py-1 text-sm",
    medium: "px-4 py-2",
    large: "px-6 py-3 text-lg"
  };
  
  const widthClass = fullWidth ? "w-full" : "";
  
  const disabledClass = (disabled || isLoading) ? "opacity-50 cursor-not-allowed" : "";
  
  const buttonClasses = `${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${widthClass} ${disabledClass} ${className}`;

  return (
    <button 
      className={buttonClasses}
      disabled={disabled || isLoading}
      {...rest}
    >
      {isLoading ? (
        <div className="flex items-center justify-center">
          <div className="animate-spin h-4 w-4 border-t-2 border-b-2 border-current mr-2"></div>
          <span>{loadingText || children}</span>
        </div>
      ) : (
        <>
          {icon && <span className="mr-2">{icon}</span>}
          {children}
        </>
      )}
    </button>
  );
};

export default Button;