/**
 * Rental Keys Administration Page (Admin Only)
 * Provides full CRUD management for rental API keys.
 */

import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { PageLayout } from '@/layouts/PageLayout';
import { RentalKeyAdminList } from '@/components/admin/rental-keys';

export const RentalKeysAdminPage: React.FC = () => {
  const navigate = useNavigate();
  const { isAdmin } = useAuthStore();

  // Redirect if not admin
  useEffect(() => {
    if (!isAdmin()) {
      navigate('/');
    }
  }, [isAdmin, navigate]);

  return (
    <PageLayout title="Rental Keys Administration">
      <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
        <div className="max-w-7xl mx-auto">
          <RentalKeyAdminList />
        </div>
      </div>
    </PageLayout>
  );
};

export default RentalKeysAdminPage;
