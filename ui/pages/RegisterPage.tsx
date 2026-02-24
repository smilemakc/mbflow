/**
 * Registration Page
 */

import React from 'react';
import {Navigate} from 'react-router-dom';
import {RegisterForm} from '@/components/auth/RegisterForm';
import {useAuthStore} from '@/store/authStore';

export const RegisterPage: React.FC = () => {
    const {isAuthenticated} = useAuthStore();

    // Redirect if already authenticated
    if (isAuthenticated) {
        return <Navigate to="/" replace/>;
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4 py-12">
            <div className="w-full max-w-md">
                {/* Logo/Brand */}
                <div className="text-center mb-8">
                    <h1 className="text-3xl font-bold text-blue-600 dark:text-blue-400">MBFlow</h1>
                    <p className="text-gray-600 dark:text-gray-400 mt-2">Workflow Orchestration Engine</p>
                </div>

                <RegisterForm/>
            </div>
        </div>
    );
};

export default RegisterPage;
