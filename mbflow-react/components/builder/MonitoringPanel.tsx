import React, {useState, useRef, useEffect} from 'react';
import {useDagStore} from '@/store/dagStore';
import {useTranslation} from '@/store/translations';
import {useReactFlow} from 'reactflow';
import {Activity, ArrowRight, ChevronDown, ChevronUp, Terminal, Trash2, Wifi, WifiOff} from 'lucide-react';
import {NodeStatus} from '@/types';
import {Button} from '@/components/ui';

export const MonitoringPanel: React.FC = () => {
    const [isExpanded, setIsExpanded] = useState(false);
    const [activeTab, setActiveTab] = useState<'logs' | 'io'>('logs');
    const {setCenter} = useReactFlow();
    const t = useTranslation();
    const logsContainerRef = useRef<HTMLDivElement>(null);

    const {
        logs,
        executionResults,
        selectedNodeId,
        clearExecution,
        nodes,
        setSelectedNodeId,
        isRunning,
        wsConnected
    } = useDagStore();

    // Auto-scroll logs to bottom when new logs arrive
    useEffect(() => {
        if (logsContainerRef.current && activeTab === 'logs') {
            logsContainerRef.current.scrollTop = logsContainerRef.current.scrollHeight;
        }
    }, [logs, activeTab]);

    const selectedNodeResult = selectedNodeId ? executionResults[selectedNodeId] : null;
    const selectedNodeLabel = selectedNodeId ? nodes.find(n => n.id === selectedNodeId)?.data.label : t.common.unknown;

    const toggleExpand = () => setIsExpanded(!isExpanded);

    // Zoom to node on log click
    const handleLogClick = (nodeId: string | null) => {
        if (nodeId) {
            setSelectedNodeId(nodeId);
            const node = nodes.find(n => n.id === nodeId);
            if (node) {
                setCenter(node.position.x + 100, node.position.y + 50, {zoom: 1.2, duration: 800});
            }
        }
    };

    if (!isExpanded) {
        return (
            <div
                className="absolute bottom-4 right-4 z-20 flex items-center bg-slate-900 text-slate-200 px-4 py-2 rounded-lg shadow-xl cursor-pointer hover:bg-slate-800 border border-slate-700"
                onClick={toggleExpand}
            >
                <Activity size={16} className="mr-2 text-blue-400"/>
                <span className="text-xs font-bold">{t.sidebar.monitoring}</span>
                {logs.length > 0 && (
                    <span className="ml-2 px-1.5 py-0.5 bg-blue-600 text-[10px] rounded-full text-white">
            {logs.length}
          </span>
                )}
                <ChevronUp size={14} className="ml-3"/>
            </div>
        );
    }

    return (
        <div
            className="absolute bottom-0 left-0 right-0 h-64 bg-white dark:bg-slate-950 border-t border-slate-200 dark:border-slate-800 z-30 flex flex-col shadow-[0_-5px_15px_rgba(0,0,0,0.1)] transition-all duration-300">

            {/* Header */}
            <div
                className="flex items-center justify-between px-4 py-2 bg-slate-50 dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 shrink-0">
                <div className="flex items-center space-x-4">
                    <div className="flex items-center text-slate-700 dark:text-slate-200 font-bold text-sm">
                        <Activity size={16} className="mr-2 text-blue-500"/>
                        {t.monitoring.title}
                    </div>

                    <div className="flex space-x-1 bg-slate-200 dark:bg-slate-800 p-0.5 rounded-md">
                        <button
                            onClick={() => setActiveTab('logs')}
                            className={`px-3 py-1 text-xs font-medium rounded-sm transition-colors ${
                                activeTab === 'logs'
                                    ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100 shadow-sm'
                                    : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                            }`}
                        >
                            {t.monitoring.consoleLogs}
                        </button>
                        <button
                            onClick={() => setActiveTab('io')}
                            className={`px-3 py-1 text-xs font-medium rounded-sm transition-colors flex items-center ${
                                activeTab === 'io'
                                    ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100 shadow-sm'
                                    : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
                            }`}
                        >
                            {t.monitoring.io}
                            {selectedNodeId && (
                                <span
                                    className="ml-2 text-[10px] opacity-60 bg-slate-300 dark:bg-slate-600 px-1 rounded">
                   {t.monitoringPanel.selected}
                </span>
                            )}
                        </button>
                    </div>
                </div>

                <div className="flex items-center space-x-2">
                    {/* WebSocket Status Indicator */}
                    {isRunning && (
                        <div className={`flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium ${
                            wsConnected
                                ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                                : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                        }`}>
                            {wsConnected ? (
                                <>
                                    <Wifi size={12} className="animate-pulse" />
                                    <span>{t.monitoringPanel.live}</span>
                                </>
                            ) : (
                                <>
                                    <WifiOff size={12} />
                                    <span>{t.monitoringPanel.offline}</span>
                                </>
                            )}
                        </div>
                    )}
                    <Button
                        variant="ghost"
                        size="sm"
                        icon={<Trash2 size={14} />}
                        onClick={clearExecution}
                        title={t.monitoring.clear}
                    />
                    <Button
                        variant="ghost"
                        size="sm"
                        icon={<ChevronDown size={14} />}
                        onClick={toggleExpand}
                    />
                </div>
            </div>

            {/* Content */}
            <div className="flex-1 overflow-hidden flex font-mono text-xs">

                {/* LOGS TAB */}
                {activeTab === 'logs' && (
                    <div ref={logsContainerRef} className="flex-1 overflow-y-auto p-4 space-y-1.5 bg-white dark:bg-slate-950">
                        {logs.length === 0 ? (
                            <div className="h-full flex flex-col items-center justify-center text-slate-400 space-y-2">
                                <Terminal size={32} className="opacity-20"/>
                                <p>{t.monitoring.ready}</p>
                            </div>
                        ) : (
                            logs.map((log) => (
                                <div
                                    key={log.id}
                                    onClick={() => handleLogClick(log.nodeId)}
                                    className={`flex items-start space-x-3 group hover:bg-slate-50 dark:hover:bg-slate-900/50 p-0.5 -mx-2 px-2 rounded cursor-pointer transition-colors ${selectedNodeId === log.nodeId ? 'bg-blue-50 dark:bg-blue-900/10' : ''}`}
                                    title={log.nodeId ? "Click to view node" : ""}
                                >
                  <span className="text-slate-400 shrink-0 select-none w-16">
                    {log.timestamp.toLocaleTimeString([], {
                        hour12: false,
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit'
                    })}
                  </span>
                                    <div className="flex-1 break-words flex items-center">
                    <span className={`w-2 h-2 rounded-full mr-2 shrink-0 ${
                        log.level === 'info' ? 'bg-blue-400' :
                            log.level === 'success' ? 'bg-green-500' :
                                log.level === 'error' ? 'bg-red-500' : 'bg-yellow-500'
                    }`}/>
                                        <span className={`
                      ${log.level === 'error' ? 'text-red-600 dark:text-red-400' :
                                            log.level === 'success' ? 'text-green-600 dark:text-green-400' :
                                                'text-slate-700 dark:text-slate-300'}
                    `}>
                      {log.message}
                    </span>
                                    </div>
                                </div>
                            ))
                        )}
                    </div>
                )}

                {/* I/O TAB */}
                {activeTab === 'io' && (
                    <div className="flex-1 flex overflow-hidden">
                        {selectedNodeResult ? (
                            <div className="flex-1 grid grid-cols-2 divide-x divide-slate-200 dark:divide-slate-800">
                                {/* Inputs */}
                                <div className="flex flex-col overflow-hidden bg-slate-50 dark:bg-slate-900/30">
                                    <div
                                        className="px-4 py-2 border-b border-slate-200 dark:border-slate-800 text-xs font-bold text-slate-500 uppercase flex justify-between items-center">
                                        <span>{t.monitoring.inputs}</span>
                                        <span
                                            className="text-[10px] bg-slate-200 dark:bg-slate-800 px-1.5 py-0.5 rounded text-slate-600 dark:text-slate-400">
                      Node: {selectedNodeLabel}
                    </span>
                                    </div>
                                    <div className="flex-1 overflow-y-auto p-4">
                     <pre className="text-slate-700 dark:text-slate-300 whitespace-pre-wrap">
                       {JSON.stringify(selectedNodeResult.inputs, null, 2)}
                     </pre>
                                    </div>
                                </div>

                                {/* Outputs */}
                                <div className="flex flex-col overflow-hidden bg-white dark:bg-slate-950">
                                    <div
                                        className="px-4 py-2 border-b border-slate-200 dark:border-slate-800 text-xs font-bold text-slate-500 uppercase flex justify-between items-center">
                                        <span>{t.monitoring.outputs}</span>
                                        <div className="flex items-center gap-2">
                       <span className={`text-[10px] px-1.5 py-0.5 rounded font-bold uppercase ${
                           selectedNodeResult.status === NodeStatus.SUCCESS ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                               selectedNodeResult.status === NodeStatus.ERROR ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' :
                                   'bg-slate-100 text-slate-600'
                       }`}>
                         {selectedNodeResult.status}
                       </span>
                                            <span className="text-[10px] text-slate-400">
                         {selectedNodeResult.endTime && selectedNodeResult.endTime - selectedNodeResult.startTime}ms
                       </span>
                                        </div>
                                    </div>
                                    <div className="flex-1 overflow-y-auto p-4">
                    <pre className="text-blue-600 dark:text-blue-400 whitespace-pre-wrap">
                       {JSON.stringify(selectedNodeResult.outputs, null, 2)}
                    </pre>
                                    </div>
                                </div>
                            </div>
                        ) : (
                            <div
                                className="flex-1 flex flex-col items-center justify-center text-slate-400 bg-slate-50 dark:bg-slate-900/50">
                                <ArrowRight size={24} className="mb-2 opacity-20"/>
                                <p>{t.monitoring.selectNode}</p>
                                {(!logs.length) &&
                                    <p className="text-[10px] mt-1 opacity-50">({t.monitoring.ready})</p>}
                            </div>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
};