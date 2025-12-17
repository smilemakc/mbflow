/**
 * RegisterForm Component
 * Handles new user registration
 */

import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useTranslations } from '@/store/translations';
import { FormField, TextInput } from '@/components/ui/form';
import { Button } from '@/components/ui';
import { configStyles } from '@/styles/configStyles';

export const RegisterForm: React.FC = () => {
  const navigate = useNavigate();
  const { register, isLoading, error, clearError } = useAuthStore();
  const t = useTranslations();

  const [formData, setFormData] = useState({
    email: '',
    password: '',
    confirmPassword: '',
    fullName: '',
  });
  const [validationError, setValidationError] = useState<string | null>(null);

  const handleFieldChange = (field: string) => (value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    setValidationError(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    clearError();
    setValidationError(null);

    // Validate passwords match
    if (formData.password !== formData.confirmPassword) {
      setValidationError(t('auth.passwordsDoNotMatch', 'Passwords do not match'));
      return;
    }

    // Validate password length
    if (formData.password.length < 8) {
      setValidationError(t('auth.passwordTooShort', 'Password must be at least 8 characters'));
      return;
    }

    const success = await register({
      email: formData.email,
      password: formData.password,
      full_name: formData.fullName || undefined,
    });

    if (success) {
      navigate('/');
    }
  };

  const displayError = validationError || error;

  return (
    <div className="w-full max-w-md mx-auto">
      <div className="bg-white dark:bg-gray-800 shadow-lg rounded-lg p-8">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            {t('auth.signUp', 'Sign Up')}
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-2">
            {t('auth.signUpDescription', 'Create a new account to get started')}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          {displayError && (
            <div className={configStyles.authError}>
              <p className="text-sm text-red-600 dark:text-red-400">{displayError}</p>
            </div>
          )}

          <FormField
            label={t('auth.email', 'Email')}
            htmlFor="email"
            required
            labelClassName={configStyles.authLabel}
          >
            <TextInput
              id="email"
              name="email"
              type="email"
              value={formData.email}
              onChange={handleFieldChange('email')}
              required
              autoComplete="email"
              placeholder="you@example.com"
              className={configStyles.authInput}
            />
          </FormField>

          <FormField
            label={t('auth.fullName', 'Full Name')}
            htmlFor="fullName"
            labelClassName={configStyles.authLabel}
          >
            <TextInput
              id="fullName"
              name="fullName"
              value={formData.fullName}
              onChange={handleFieldChange('fullName')}
              autoComplete="name"
              placeholder="John Doe"
              className={configStyles.authInput}
            />
          </FormField>

          <FormField
            label={t('auth.password', 'Password')}
            htmlFor="password"
            required
            hint={t('auth.passwordHint', 'At least 8 characters with uppercase, lowercase, and number')}
            labelClassName={configStyles.authLabel}
          >
            <TextInput
              id="password"
              name="password"
              type="password"
              value={formData.password}
              onChange={handleFieldChange('password')}
              required
              autoComplete="new-password"
              minLength={8}
              placeholder="********"
              className={configStyles.authInput}
            />
          </FormField>

          <FormField
            label={t('auth.confirmPassword', 'Confirm Password')}
            htmlFor="confirmPassword"
            required
            labelClassName={configStyles.authLabel}
          >
            <TextInput
              id="confirmPassword"
              name="confirmPassword"
              type="password"
              value={formData.confirmPassword}
              onChange={handleFieldChange('confirmPassword')}
              required
              autoComplete="new-password"
              placeholder="********"
              className={configStyles.authInput}
            />
          </FormField>

          <Button
            type="submit"
            loading={isLoading}
            fullWidth
            size="lg"
          >
            {isLoading
              ? t('auth.creatingAccount', 'Creating account...')
              : t('auth.createAccount', 'Create Account')}
          </Button>
        </form>

        <div className="mt-6 text-center">
          <p className="text-gray-600 dark:text-gray-400">
            {t('auth.alreadyHaveAccount', 'Already have an account?')}{' '}
            <Link
              to="/login"
              className="text-blue-600 hover:text-blue-700 dark:text-blue-400 font-medium"
            >
              {t('auth.signIn', 'Sign In')}
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
};

export default RegisterForm;
