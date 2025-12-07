import React, {useEffect} from 'react';
import {HashRouter, Navigate, Route, Routes} from 'react-router-dom';
import {useUIStore} from './store/uiStore';
import {ToastContainer} from '@/components/ui';

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

const App: React.FC = () => {
    const {theme} = useUIStore();

    useEffect(() => {
        const root = window.document.documentElement;
        root.classList.remove('light', 'dark');
        root.classList.add(theme);
    }, [theme]);

    return (
        <HashRouter>
            <Routes>
                {/* Default redirect */}
                <Route path="/" element={<Navigate to="/builder" replace/>}/>

                {/* Builder (complex layout with ReactFlow) */}
                <Route path="/builder" element={<BuilderLayout/>}/>

                {/* Standard pages with simple layout */}
                <Route path="/dashboard" element={
                    <PageLayout title="Dashboard"><DashboardPage/></PageLayout>
                }/>
                <Route path="/workflows" element={
                    <PageLayout title="Workflows"><WorkflowsPage/></PageLayout>
                }/>
                <Route path="/executions" element={
                    <PageLayout title="Execution History"><ExecutionsPage/></PageLayout>
                }/>
                <Route path="/executions/:id" element={
                    <PageLayout title="Execution Details"><ExecutionDetailPage/></PageLayout>
                }/>
                <Route path="/triggers" element={
                    <PageLayout title="Triggers"><TriggersPage/></PageLayout>
                }/>
                <Route path="/monitoring" element={
                    <PageLayout title="System Monitoring"><MonitoringPage/></PageLayout>
                }/>
                <Route path="/resources" element={
                    <PageLayout title="Resources"><ResourcesPage/></PageLayout>
                }/>
                <Route path="/settings" element={
                    <PageLayout title="Settings"><SettingsPage/></PageLayout>
                }/>

                {/* Fallback */}
                <Route path="*" element={<Navigate to="/builder" replace/>}/>
            </Routes>
            <ToastContainer/>
        </HashRouter>
    );
};

export default App;
