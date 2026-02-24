import React from 'react';
import {Activity, AlertTriangle, Cpu, Download, Filter, HardDrive, Search, Server, Terminal} from 'lucide-react';
import {Button} from '@/components/ui';

export const MonitoringPage: React.FC = () => {
    return (
        <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
            <div className="max-w-7xl mx-auto space-y-6">

                {/* Header */}
                <div className="flex justify-between items-end">
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">System Monitoring</h1>
                        <p className="text-slate-500 dark:text-slate-400 mt-1">Real-time infrastructure performance and
                            error tracking.</p>
                    </div>
                    <div className="flex space-x-3">
                        <Button
                            variant="outline"
                            size="sm"
                            icon={<Filter size={16}/>}
                        >
                            Filter
                        </Button>
                        <Button
                            variant="primary"
                            size="sm"
                            icon={<Download size={16}/>}
                        >
                            Export Logs
                        </Button>
                    </div>
                </div>

                {/* Live Metrics */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    <MetricCard
                        title="CPU Usage"
                        value="42%"
                        subtext="8 Cores Active"
                        icon={<Cpu size={20}/>}
                        color="blue"
                        graph={[20, 35, 45, 40, 42, 48, 42]}
                    />
                    <MetricCard
                        title="Memory Usage"
                        value="3.2 GB"
                        subtext="of 8.0 GB Total"
                        icon={<HardDrive size={20}/>}
                        color="purple"
                        graph={[60, 62, 61, 65, 64, 63, 62]}
                    />
                    <MetricCard
                        title="Active Workers"
                        value="12/16"
                        subtext="4 Idle"
                        icon={<Server size={20}/>}
                        color="green"
                        graph={[10, 10, 12, 12, 11, 12, 12]}
                    />
                </div>

                {/* Main Content Area */}
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">

                    {/* Error Log / Events */}
                    <div
                        className="lg:col-span-2 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm flex flex-col h-[600px]">
                        <div
                            className="p-4 border-b border-slate-200 dark:border-slate-800 flex justify-between items-center bg-slate-50/50 dark:bg-slate-900/50">
                            <h3 className="font-bold text-slate-800 dark:text-slate-100 flex items-center">
                                <Terminal size={16} className="mr-2 text-slate-500"/>
                                System Event Log
                            </h3>
                            <div className="relative">
                                <Search size={14} className="absolute left-2.5 top-2.5 text-slate-400"/>
                                <input
                                    type="text"
                                    placeholder="Search logs..."
                                    className="pl-8 pr-3 py-1.5 text-xs bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-700 rounded-md focus:outline-none focus:border-blue-500 text-slate-700 dark:text-slate-300 w-48"
                                />
                            </div>
                        </div>
                        <div className="flex-1 overflow-y-auto p-2 space-y-1 font-mono text-xs">
                            {MOCK_LOGS.map((log, i) => (
                                <div key={i}
                                     className={`flex items-start p-2 rounded hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer ${
                                         log.level === 'error' ? 'bg-red-50 dark:bg-red-900/10' : ''
                                     }`}>
                                    <span className="text-slate-400 w-24 shrink-0">{log.time}</span>
                                    <span className={`w-16 font-bold shrink-0 ${
                                        log.level === 'info' ? 'text-blue-500' :
                                            log.level === 'warn' ? 'text-orange-500' :
                                                'text-red-500'
                                    }`}>{log.level.toUpperCase()}</span>
                                    <span className="text-slate-700 dark:text-slate-300 break-all">{log.msg}</span>
                                </div>
                            ))}
                        </div>
                    </div>

                    {/* Health Status */}
                    <div
                        className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm p-5 space-y-6">
                        <div>
                            <h3 className="font-bold text-slate-800 dark:text-slate-100 mb-4 flex items-center">
                                <Activity size={18} className="mr-2 text-slate-400"/>
                                Service Health
                            </h3>
                            <div className="space-y-3">
                                <ServiceStatus name="API Gateway" status="operational" uptime="99.99%"/>
                                <ServiceStatus name="Workflow Engine" status="operational" uptime="99.95%"/>
                                <ServiceStatus name="Database (Primary)" status="operational" uptime="99.99%"/>
                                <ServiceStatus name="Redis Cache" status="degraded" uptime="98.50%"/>
                                <ServiceStatus name="Notification Svc" status="operational" uptime="100%"/>
                            </div>
                        </div>

                        <div className="pt-6 border-t border-slate-100 dark:border-slate-800">
                            <h3 className="font-bold text-slate-800 dark:text-slate-100 mb-2">Incidents</h3>
                            <div
                                className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-900/30 rounded-lg p-3">
                                <div className="flex items-start">
                                    <AlertTriangle size={16}
                                                   className="text-yellow-600 dark:text-yellow-500 mt-0.5 mr-2 shrink-0"/>
                                    <div>
                                        <h4 className="text-sm font-bold text-yellow-800 dark:text-yellow-500">High
                                            Latency Detected</h4>
                                        <p className="text-xs text-yellow-700 dark:text-yellow-400/80 mt-1">Redis cache
                                            response time 200ms. Investigating.</p>
                                        <p className="text-[10px] text-yellow-600/60 mt-2">Started 15 mins ago</p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

            </div>
        </div>
    );
};

const MetricCard = ({title, value, subtext, icon, color, graph}: any) => (
    <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 shadow-sm">
        <div className="flex justify-between items-start mb-4">
            <div>
                <h3 className="text-slate-500 dark:text-slate-400 text-sm font-medium">{title}</h3>
                <div className="text-2xl font-bold text-slate-900 dark:text-white mt-1">{value}</div>
                <div className="text-xs text-slate-400 mt-0.5">{subtext}</div>
            </div>
            <div
                className={`p-2 bg-${color}-50 dark:bg-${color}-900/20 text-${color}-600 dark:text-${color}-400 rounded-lg`}>
                {icon}
            </div>
        </div>
        {/* Mock Mini Graph */}
        <div className="flex items-end space-x-1 h-8">
            {graph.map((h: number, i: number) => (
                <div
                    key={i}
                    className={`flex-1 bg-${color}-100 dark:bg-${color}-900/40 rounded-sm`}
                    style={{height: `${h}%`}}
                ></div>
            ))}
        </div>
    </div>
);

const ServiceStatus = ({name, status, uptime}: any) => (
    <div className="flex items-center justify-between text-sm">
        <div className="flex items-center">
            <div className={`w-2 h-2 rounded-full mr-2 ${
                status === 'operational' ? 'bg-green-500' :
                    status === 'degraded' ? 'bg-yellow-500' : 'bg-red-500'
            }`}></div>
            <span className="text-slate-700 dark:text-slate-300">{name}</span>
        </div>
        <span className="text-slate-400 text-xs font-mono">{uptime}</span>
    </div>
);

const MOCK_LOGS = [
    {time: '10:42:01.320', level: 'info', msg: 'Worker-01 connected successfully.'},
    {time: '10:42:05.112', level: 'info', msg: 'Job #8492 started processing.'},
    {time: '10:42:08.540', level: 'warn', msg: 'API Rate limit approaching (90%).'},
    {time: '10:42:12.001', level: 'info', msg: 'Job #8492 completed in 6.8s.'},
    {time: '10:43:45.220', level: 'error', msg: 'Connection timeout: PostgreSQL pool full.'},
    {time: '10:43:45.350', level: 'info', msg: 'Retrying connection (Attempt 1/3)...'},
    {time: '10:43:46.002', level: 'info', msg: 'Connection established.'},
    {time: '10:44:10.100', level: 'info', msg: 'Scheduled maintenance check completed.'},
    {time: '10:44:15.550', level: 'warn', msg: 'High memory usage detected on Worker-03.'},
    {time: '10:45:00.000', level: 'info', msg: 'Cron trigger [DailyCleanup] fired.'},
];
