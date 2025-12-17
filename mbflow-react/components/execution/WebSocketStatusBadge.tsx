import React from 'react';
import {Wifi, WifiOff} from 'lucide-react';

interface WebSocketStatusBadgeProps {
    connected: boolean;
}

export const WebSocketStatusBadge: React.FC<WebSocketStatusBadgeProps> = ({connected}) => {
    return (
        <div className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium ${
            connected
                ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
        }`}>
            {connected ? (
                <>
                    <Wifi size={14} className="animate-pulse"/>
                    <span>Live Updates</span>
                </>
            ) : (
                <>
                    <WifiOff size={14}/>
                    <span>Connecting...</span>
                </>
            )}
        </div>
    );
};
