/**
 * LoginForm Component
 * Handles user login with email and password
 */

import React, { useState } from 'react';
import { useNavigate, useLocation, Link } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useTranslations } from '@/store/translations';
import { FormField, TextInput } from '@/components/ui/form';
import { Button } from '@/components/ui';
import { configStyles } from '@/styles/configStyles';

export const LoginForm: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { login, isLoading, error, clearError } = useAuthStore();
  const t = useTranslations();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const from = (location.state as any)?.from?.pathname || '/';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    clearError();

    const success = await login({ email, password });
    if (success) {
      navigate(from, { replace: true });
    }
  };

  return (
    <div className="w-full max-w-md mx-auto">
      <div className="bg-white dark:bg-gray-800 shadow-lg rounded-lg p-8">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            {t('auth.signIn', 'Sign In')}
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-2">
            {t('auth.signInDescription', 'Enter your credentials to access your account')}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          {error && (
            <div className={configStyles.authError}>
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            </div>
          )}

          <FormField
            label={t('auth.email', 'Email')}
            htmlFor="email"
            labelClassName={configStyles.authLabel}
          >
            <TextInput
              id="email"
              type="email"
              value={email}
              onChange={setEmail}
              required
              autoComplete="email"
              placeholder="you@example.com"
              className={configStyles.authInput}
            />
          </FormField>

          <FormField
            label={t('auth.password', 'Password')}
            htmlFor="password"
            labelClassName={configStyles.authLabel}
          >
            <TextInput
              id="password"
              type="password"
              value={password}
              onChange={setPassword}
              required
              autoComplete="current-password"
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
            {isLoading ? t('auth.signingIn', 'Signing in...') : t('auth.signIn', 'Sign In')}
          </Button>
        </form>

        <div className="mt-6 text-center">
          <p className="text-gray-600 dark:text-gray-400">
            {t('auth.noAccount', "Don't have an account?")}{' '}
            <Link
              to="/register"
              className="text-blue-600 hover:text-blue-700 dark:text-blue-400 font-medium"
            >
              {t('auth.signUp', 'Sign Up')}
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
};

export default LoginForm;
