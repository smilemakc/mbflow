/**
 * UserSelector component
 * Searchable dropdown for selecting users with avatar display.
 */

import React, { useState, useRef, useEffect, useMemo } from 'react';
import { Search, ChevronDown, X, User as UserIcon } from 'lucide-react';
import type { User } from '@/types/auth';

interface UserSelectorProps {
  value: User | null;
  onChange: (user: User | null) => void;
  users: User[];
  loading?: boolean;
  placeholder?: string;
  className?: string;
  disabled?: boolean;
  label?: string;
  required?: boolean;
  showClearButton?: boolean;
}

export const UserSelector: React.FC<UserSelectorProps> = ({
  value,
  onChange,
  users,
  loading = false,
  placeholder = 'Select user...',
  className = '',
  disabled = false,
  label,
  required = false,
  showClearButton = true,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setSearch('');
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const filteredUsers = useMemo(() => {
    if (!search.trim()) return users;
    const searchLower = search.toLowerCase();
    return users.filter(
      (user) =>
        user.username.toLowerCase().includes(searchLower) ||
        user.email.toLowerCase().includes(searchLower) ||
        (user.full_name && user.full_name.toLowerCase().includes(searchLower))
    );
  }, [users, search]);

  const handleSelect = (user: User) => {
    onChange(user);
    setIsOpen(false);
    setSearch('');
  };

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation();
    onChange(null);
  };

  const getUserInitials = (user: User): string => {
    if (user.full_name) {
      const parts = user.full_name.trim().split(' ');
      if (parts.length >= 2) {
        return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
      }
    }
    return user.username.slice(0, 2).toUpperCase();
  };

  return (
    <div ref={dropdownRef} className={`relative ${className}`}>
      {label && (
        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
          {label}
          {required && <span className="text-red-500 ml-1">*</span>}
        </label>
      )}

      <button
        type="button"
        onClick={() => !disabled && setIsOpen(!isOpen)}
        disabled={disabled}
        className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-slate-800 text-left flex items-center justify-between transition-colors ${
          disabled
            ? 'opacity-50 cursor-not-allowed'
            : isOpen
            ? 'border-blue-500 ring-2 ring-blue-500/20'
            : 'border-slate-200 dark:border-slate-700 hover:border-slate-300 dark:hover:border-slate-600'
        }`}
      >
        {value ? (
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <div className="w-6 h-6 rounded-full bg-blue-500 flex items-center justify-center text-white text-xs font-medium flex-shrink-0">
              {getUserInitials(value)}
            </div>
            <div className="min-w-0 flex-1">
              <span className="text-slate-900 dark:text-white truncate block text-sm">
                {value.full_name || value.username}
              </span>
              <span className="text-xs text-slate-500 dark:text-slate-400 truncate block">
                @{value.username}
              </span>
            </div>
          </div>
        ) : (
          <span className="text-slate-400 dark:text-slate-500 text-sm">
            {placeholder}
          </span>
        )}
        <div className="flex items-center gap-1 flex-shrink-0">
          {value && showClearButton && !disabled && (
            <button
              type="button"
              onClick={handleClear}
              className="p-0.5 hover:bg-slate-100 dark:hover:bg-slate-700 rounded"
            >
              <X size={14} className="text-slate-400" />
            </button>
          )}
          <ChevronDown
            size={16}
            className={`text-slate-400 transition-transform ${isOpen ? 'rotate-180' : ''}`}
          />
        </div>
      </button>

      {isOpen && (
        <div className="absolute z-50 w-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg max-h-72 overflow-hidden">
          <div className="p-2 border-b border-slate-200 dark:border-slate-700">
            <div className="relative">
              <Search
                size={14}
                className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400"
              />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search users..."
                className="w-full pl-8 pr-3 py-1.5 text-sm border border-slate-200 dark:border-slate-700 rounded bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                autoFocus
              />
            </div>
          </div>

          <div className="max-h-52 overflow-y-auto">
            {loading ? (
              <div className="p-3 text-center text-slate-500 dark:text-slate-400 text-sm">
                <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto mb-1" />
                Loading...
              </div>
            ) : filteredUsers.length === 0 ? (
              <div className="p-3 text-center text-slate-500 dark:text-slate-400 text-sm">
                No users found
              </div>
            ) : (
              filteredUsers.map((user) => (
                <button
                  key={user.id}
                  type="button"
                  onClick={() => handleSelect(user)}
                  className={`w-full px-3 py-2 flex items-center gap-2 hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors ${
                    value?.id === user.id ? 'bg-blue-50 dark:bg-blue-900/20' : ''
                  }`}
                >
                  <div className="w-6 h-6 rounded-full bg-blue-500 flex items-center justify-center text-white text-xs font-medium flex-shrink-0">
                    {getUserInitials(user)}
                  </div>
                  <div className="flex-1 min-w-0 text-left">
                    <div className="text-sm font-medium text-slate-900 dark:text-white truncate">
                      {user.full_name || user.username}
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400 truncate">
                      @{user.username} • {user.email}
                    </div>
                  </div>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default UserSelector;
