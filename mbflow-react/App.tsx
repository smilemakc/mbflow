import React, {useEffect} from 'react';
import {HashRouter, Navigate, Route, Routes} from 'react-router-dom';
import {useUIStore} from './store/uiStore';
import {useAuthStore, initializeAuth} from './store/authStore';
import {ToastContainer} from '@/components/ui';
import {ProtectedRoute} from '@/components/auth';

// Layouts
import {BuilderLayout, PageLayout} from '@/layouts';

// Pages
import {
    DashboardPage,
    ExecutionsPage,
    ExecutionDetailPage,
    MonitoringPage,
    ResourcesPage,
    SettingsPage,
    TriggersPage,
    WorkflowsPage
} from './pages';

// Auth Pages
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import UsersPage from './pages/UsersPage';

const App: React.FC = () => {
    const {theme} = useUIStore();
    const {isInitialized} = useAuthStore();

    // Initialize theme
    useEffect(() => {
        const root = window.document.documentElement;
        root.classList.remove('light', 'dark');
        root.classList.add(theme);
    }, [theme]);

    // Initialize auth on app load
    useEffect(() => {
        initializeAuth();
    }, []);

    // Show loading while auth is initializing
    if (!isInitialized) {
        return (
            <div className="flex items-center justify-center min-h-screen bg-gray-50 dark:bg-gray-900">
                <div className="flex flex-col items-center gap-4">
                    <div className="w-8 h-8 border-4 border-blue-500 border-t-transparent rounded-full animate-spin" />
                    <p className="text-gray-600 dark:text-gray-400">Loading...</p>
                </div>
            </div>
        );
    }

    return (
        <HashRouter>
            <Routes>
                {/* Public auth routes */}
                <Route path="/login" element={<LoginPage/>}/>
                <Route path="/register" element={<RegisterPage/>}/>

                {/* Default redirect */}
                <Route path="/" element={<Navigate to="/builder" replace/>}/>

                {/* Protected routes */}
                <Route path="/builder" element={
                    <ProtectedRoute><BuilderLayout/></ProtectedRoute>
                }/>

                {/* Standard pages with simple layout */}
                <Route path="/dashboard" element={
                    <ProtectedRoute>
                        <PageLayout title="Dashboard"><DashboardPage/></PageLayout>
                    </ProtectedRoute>
                }/>
                <Route path="/workflows" element={
                    <ProtectedRoute>
                        <PageLayout title="Workflows"><WorkflowsPage/></PageLayout>
                    </ProtectedRoute>
                }/>
                <Route path="/executions" element={
                    <ProtectedRoute>
                        <PageLayout title="Execution History"><ExecutionsPage/></PageLayout>
                    </ProtectedRoute>
                }/>
                <Route path="/executions/:id" element={
                    <ProtectedRoute>
                        <PageLayout title="Execution Details"><ExecutionDetailPage/></PageLayout>
                    </ProtectedRoute>
                }/>
                <Route path="/triggers" element={
                    <ProtectedRoute>
                        <PageLayout title="Triggers"><TriggersPage/></PageLayout>
                    </ProtectedRoute>
                }/>
                <Route path="/monitoring" element={
                    <ProtectedRoute>
                        <PageLayout title="System Monitoring"><MonitoringPage/></PageLayout>
                    </ProtectedRoute>
                }/>
                <Route path="/resources" element={
                    <ProtectedRoute>
                        <PageLayout title="Resources"><ResourcesPage/></PageLayout>
                    </ProtectedRoute>
                }/>
                <Route path="/settings" element={
                    <ProtectedRoute>
                        <PageLayout title="Settings"><SettingsPage/></PageLayout>
                    </ProtectedRoute>
                }/>

                {/* Admin routes */}
                <Route path="/admin/users" element={
                    <ProtectedRoute requireAdmin>
                        <UsersPage/>
                    </ProtectedRoute>
                }/>

                {/* Fallback */}
                <Route path="*" element={<Navigate to="/builder" replace/>}/>
            </Routes>
            <ToastContainer/>
        </HashRouter>
    );
};

export default App;
