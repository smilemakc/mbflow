import React, {useEffect, useState} from 'react';
import {Activity, AlertCircle, CheckCircle2, Clock, Server, TrendingUp, Workflow, Zap} from 'lucide-react';
import {useDagStore} from '@/store/dagStore';
import {useUIStore} from '@/store/uiStore';
import {useTranslation} from '@/store/translations';
import {useNavigate} from 'react-router-dom';
import {workflowService} from '@/services/workflowService';
import {executionService} from '@/services/executionService';
import {Button} from '@/components/ui';

interface RecentRun {
    id: string;
    name: string;
    status: 'success' | 'error' | 'running' | 'pending';
    time: string;
    duration: string;
    user: string;
}

export const DashboardPage: React.FC = () => {
    const {nodes} = useDagStore();
    const {setActiveModal} = useUIStore();
    const navigate = useNavigate();
    const t = useTranslation();

    const [workflowCount, setWorkflowCount] = useState<number>(0);
    const [recentRuns, setRecentRuns] = useState<RecentRun[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        const fetchData = async () => {
            setIsLoading(true);
            try {
                // Fetch workflow count
                const workflows = await workflowService.getAll();
                setWorkflowCount(workflows.length);

                // Fetch recent executions
                const executions = await executionService.getRecent(5);
                const runs: RecentRun[] = executions.map(exec => ({
                    id: exec.id.substring(0, 8),
                    name: exec.workflow_id.substring(0, 12) + '...',
                    status: exec.status === 'completed' ? 'success'
                        : exec.status === 'failed' ? 'error'
                            : exec.status === 'running' ? 'running' : 'pending',
                    time: formatRelativeTime(exec.started_at),
                    duration: exec.completed_at
                        ? `${((new Date(exec.completed_at).getTime() - new Date(exec.started_at).getTime()) / 1000).toFixed(1)}s`
                        : '-',
                    user: 'System'
                }));
                setRecentRuns(runs);
            } catch (error) {
                console.error('Failed to fetch dashboard data:', error);
            } finally {
                setIsLoading(false);
            }
        };

        fetchData();
    }, []);

    const handleCreateNew = () => {
        setActiveModal('templates');
        navigate('/builder');
    };

    // Stats with real workflow count
    const stats = [
        {
            label: t.dashboard.totalWorkflows,
            value: workflowCount.toString(),
            icon: Workflow,
            change: '',
            trend: 'up',
            color: 'blue'
        },
        {
            label: t.dashboard.executionsToday,
            value: recentRuns.length.toString(),
            icon: Zap,
            change: '',
            trend: 'up',
            color: 'orange'
        },
        {label: t.dashboard.successRate, value: '-', icon: CheckCircle2, change: '', trend: 'up', color: 'green'},
        {label: t.dashboard.avgDuration, value: '-', icon: Clock, change: '', trend: 'down', color: 'purple'},
    ];

    function formatRelativeTime(dateStr: string): string {
        const date = new Date(dateStr);
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffMins = Math.floor(diffMs / 60000);

        if (diffMins < 1) return 'just now';
        if (diffMins < 60) return `${diffMins} mins ago`;
        const diffHours = Math.floor(diffMins / 60);
        if (diffHours < 24) return `${diffHours} hours ago`;
        const diffDays = Math.floor(diffHours / 24);
        return `${diffDays} days ago`;
    }

    return (
        <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
            <div className="max-w-7xl mx-auto space-y-8">

                {/* Header */}
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t.dashboard.title}</h1>
                    <p className="text-slate-500 dark:text-slate-400 mt-1">{t.dashboard.subtitle}</p>
                </div>

                {/* Stats Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                    {stats.map((stat, index) => (
                        <div key={index}
                             className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-sm hover:shadow-md transition-shadow">
                            <div className="flex justify-between items-start">
                                <div
                                    className={`p-2 rounded-lg bg-${stat.color}-50 dark:bg-${stat.color}-900/20 text-${stat.color}-600 dark:text-${stat.color}-400`}>
                                    <stat.icon size={20}/>
                                </div>
                                <div
                                    className={`flex items-center text-xs font-medium ${(stat.trend === 'up' && stat.label !== t.dashboard.avgDuration) || (stat.trend === 'down' && stat.label === t.dashboard.avgDuration)
                                        ? 'text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20'
                                        : 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20'
                                    } px-2 py-0.5 rounded-full`}
                                >
                                    {stat.change}
                                    <TrendingUp size={12}
                                                className={`ml-1 ${stat.trend === 'down' ? 'rotate-180' : ''}`}/>
                                </div>
                            </div>
                            <div className="mt-4">
                                <h3 className="text-2xl font-bold text-slate-900 dark:text-white">{stat.value}</h3>
                                <p className="text-sm text-slate-500 dark:text-slate-400 font-medium">{stat.label}</p>
                            </div>
                        </div>
                    ))}
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">

                    {/* Recent Activity */}
                    <div
                        className="lg:col-span-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden flex flex-col">
                        <div
                            className="p-5 border-b border-slate-200 dark:border-slate-800 flex justify-between items-center bg-slate-50/50 dark:bg-slate-900/50">
                            <h2 className="font-bold text-slate-800 dark:text-slate-100 flex items-center">
                                <Activity size={18} className="mr-2 text-slate-400"/>
                                {t.dashboard.recentActivity}
                            </h2>
                            <Button variant="ghost" size="sm">{t.dashboard.viewAll}</Button>
                        </div>
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm text-left">
                                <thead
                                    className="text-xs text-slate-500 uppercase bg-slate-50 dark:bg-slate-900/50 border-b border-slate-100 dark:border-slate-800">
                                <tr>
                                    <th className="px-6 py-3 font-medium">{t.dashboard.table.workflow}</th>
                                    <th className="px-6 py-3 font-medium">{t.dashboard.table.status}</th>
                                    <th className="px-6 py-3 font-medium">{t.dashboard.table.duration}</th>
                                    <th className="px-6 py-3 font-medium">{t.dashboard.table.triggeredBy}</th>
                                    <th className="px-6 py-3 font-medium text-right">{t.dashboard.table.time}</th>
                                </tr>
                                </thead>
                                <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                                {recentRuns.map((run) => (
                                    <tr key={run.id}
                                        className="hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors group">
                                        <td className="px-6 py-4 font-medium text-slate-900 dark:text-slate-200">
                                            {run.name}
                                            <div
                                                className="text-[10px] text-slate-400 font-normal font-mono mt-0.5">{run.id}</div>
                                        </td>
                                        <td className="px-6 py-4">
                        <span
                            className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium border ${run.status === 'success'
                                ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-400 dark:border-green-900/30'
                                : 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-400 dark:border-red-900/30'
                            }`}>
                          {run.status === 'success' ? <CheckCircle2 size={12} className="mr-1.5"/> :
                              <AlertCircle size={12} className="mr-1.5"/>}
                            {run.status === 'success' ? t.common.success : t.common.error}
                        </span>
                                        </td>
                                        <td className="px-6 py-4 text-slate-500 dark:text-slate-400 font-mono text-xs">{run.duration}</td>
                                        <td className="px-6 py-4 text-slate-600 dark:text-slate-300">
                        <span
                            className="inline-flex items-center px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 text-xs">
                          {run.user}
                        </span>
                                        </td>
                                        <td className="px-6 py-4 text-right text-slate-500 dark:text-slate-400">
                                            {run.time}
                                        </td>
                                    </tr>
                                ))}
                                </tbody>
                            </table>
                        </div>
                    </div>

                    {/* System Health / API Usage */}
                    <div className="space-y-6">
                        <div
                            className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm p-5">
                            <h2 className="font-bold text-slate-800 dark:text-slate-100 mb-4 flex items-center">
                                <Server size={18} className="mr-2 text-slate-400"/>
                                {t.dashboard.systemHealth}
                            </h2>

                            <div className="space-y-4">
                                <div>
                                    <div className="flex justify-between text-xs mb-1.5">
                                        <span className="text-slate-500 dark:text-slate-400">API Latency (p95)</span>
                                        <span className="font-medium text-slate-700 dark:text-slate-300">142ms</span>
                                    </div>
                                    <div className="w-full bg-slate-100 dark:bg-slate-800 rounded-full h-2">
                                        <div className="bg-blue-500 h-2 rounded-full" style={{width: '35%'}}></div>
                                    </div>
                                </div>

                                <div>
                                    <div className="flex justify-between text-xs mb-1.5">
                                        <span className="text-slate-500 dark:text-slate-400">Error Rate</span>
                                        <span className="font-medium text-green-600 dark:text-green-400">0.02%</span>
                                    </div>
                                    <div className="w-full bg-slate-100 dark:bg-slate-800 rounded-full h-2">
                                        <div className="bg-green-500 h-2 rounded-full" style={{width: '2%'}}></div>
                                    </div>
                                </div>

                                <div>
                                    <div className="flex justify-between text-xs mb-1.5">
                                        <span className="text-slate-500 dark:text-slate-400">Worker Utilization</span>
                                        <span className="font-medium text-orange-600 dark:text-orange-400">78%</span>
                                    </div>
                                    <div className="w-full bg-slate-100 dark:bg-slate-800 rounded-full h-2">
                                        <div className="bg-orange-500 h-2 rounded-full" style={{width: '78%'}}></div>
                                    </div>
                                </div>
                            </div>

                            <div
                                className="mt-6 pt-5 border-t border-slate-100 dark:border-slate-800 grid grid-cols-2 gap-4">
                                <div className="bg-slate-50 dark:bg-slate-800/50 p-3 rounded-lg text-center">
                                    <div
                                        className="text-[10px] uppercase text-slate-400 font-bold tracking-wider mb-1">Queue
                                        Depth
                                    </div>
                                    <div className="text-xl font-bold text-slate-800 dark:text-white">24</div>
                                </div>
                                <div className="bg-slate-50 dark:bg-slate-800/50 p-3 rounded-lg text-center">
                                    <div
                                        className="text-[10px] uppercase text-slate-400 font-bold tracking-wider mb-1">Active
                                        Nodes
                                    </div>
                                    <div
                                        className="text-xl font-bold text-slate-800 dark:text-white">{nodes.length}</div>
                                </div>
                            </div>
                        </div>

                        {/* Quick Actions Card */}
                        <div
                            className="bg-gradient-to-br from-blue-600 to-indigo-700 rounded-xl shadow-lg p-5 text-white">
                            <h3 className="font-bold text-lg mb-2">{t.dashboard.quickActions}</h3>
                            <p className="text-blue-100 text-sm mb-4">{t.dashboard.quickDesc}</p>
                            <Button
                                onClick={handleCreateNew}
                                className="w-full bg-white text-blue-600 font-semibold hover:bg-blue-50 shadow-sm"
                            >
                                {t.dashboard.createWorkflow}
                            </Button>
                        </div>
                    </div>

                </div>
            </div>
        </div>
    );
};