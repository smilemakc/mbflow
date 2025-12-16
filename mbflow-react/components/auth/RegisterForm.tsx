/**
 * RegisterForm Component
 * Handles new user registration
 */

import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useTranslations } from '@/store/translations';

export const RegisterForm: React.FC = () => {
  const navigate = useNavigate();
  const { register, isLoading, error, clearError } = useAuthStore();
  const t = useTranslations();

  const [formData, setFormData] = useState({
    email: '',
    username: '',
    password: '',
    confirmPassword: '',
    fullName: '',
  });
  const [validationError, setValidationError] = useState<string | null>(null);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
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

    // Validate username
    if (formData.username.length < 3) {
      setValidationError(t('auth.usernameTooShort', 'Username must be at least 3 characters'));
      return;
    }

    const success = await register({
      email: formData.email,
      username: formData.username,
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
            <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
              <p className="text-sm text-red-600 dark:text-red-400">{displayError}</p>
            </div>
          )}

          <div>
            <label
              htmlFor="email"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              {t('auth.email', 'Email')} *
            </label>
            <input
              id="email"
              name="email"
              type="email"
              value={formData.email}
              onChange={handleChange}
              required
              autoComplete="email"
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white transition-colors"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label
              htmlFor="username"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              {t('auth.username', 'Username')} *
            </label>
            <input
              id="username"
              name="username"
              type="text"
              value={formData.username}
              onChange={handleChange}
              required
              autoComplete="username"
              minLength={3}
              maxLength={50}
              pattern="[a-zA-Z0-9_-]+"
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white transition-colors"
              placeholder="johndoe"
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {t('auth.usernameHint', 'Letters, numbers, underscores and hyphens only')}
            </p>
          </div>

          <div>
            <label
              htmlFor="fullName"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              {t('auth.fullName', 'Full Name')}
            </label>
            <input
              id="fullName"
              name="fullName"
              type="text"
              value={formData.fullName}
              onChange={handleChange}
              autoComplete="name"
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white transition-colors"
              placeholder="John Doe"
            />
          </div>

          <div>
            <label
              htmlFor="password"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              {t('auth.password', 'Password')} *
            </label>
            <input
              id="password"
              name="password"
              type="password"
              value={formData.password}
              onChange={handleChange}
              required
              autoComplete="new-password"
              minLength={8}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white transition-colors"
              placeholder="********"
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {t('auth.passwordHint', 'At least 8 characters with uppercase, lowercase, and number')}
            </p>
          </div>

          <div>
            <label
              htmlFor="confirmPassword"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"
            >
              {t('auth.confirmPassword', 'Confirm Password')} *
            </label>
            <input
              id="confirmPassword"
              name="confirmPassword"
              type="password"
              value={formData.confirmPassword}
              onChange={handleChange}
              required
              autoComplete="new-password"
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-white transition-colors"
              placeholder="********"
            />
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full py-3 px-4 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
          >
            {isLoading ? (
              <>
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                {t('auth.creatingAccount', 'Creating account...')}
              </>
            ) : (
              t('auth.createAccount', 'Create Account')
            )}
          </button>
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
