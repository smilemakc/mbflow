import React, {useEffect, useRef, useState} from 'react';
import {NodeType} from '@/types';
import {Bot, Clock, GitBranch, Globe, Search, Sparkles} from 'lucide-react';

interface QuickAddMenuProps {
    x: number;
    y: number;
    onClose: () => void;
    onSelect: (type: NodeType) => void;
}

const ITEMS = [
    {
        type: NodeType.HTTP,
        label: 'HTTP Request',
        icon: Globe,
        color: 'text-green-500',
        bg: 'bg-green-100 dark:bg-green-900/30'
    },
    {
        type: NodeType.LLM,
        label: 'AI Generator',
        icon: Sparkles,
        color: 'text-purple-500',
        bg: 'bg-purple-100 dark:bg-purple-900/30'
    },
    {
        type: NodeType.CONDITIONAL,
        label: 'Condition',
        icon: GitBranch,
        color: 'text-slate-500',
        bg: 'bg-slate-100 dark:bg-slate-800'
    },
    {
        type: NodeType.TELEGRAM,
        label: 'Telegram Bot',
        icon: Bot,
        color: 'text-blue-500',
        bg: 'bg-blue-100 dark:bg-blue-900/30'
    },
    {
        type: NodeType.DELAY,
        label: 'Scheduler',
        icon: Clock,
        color: 'text-orange-500',
        bg: 'bg-orange-100 dark:bg-orange-900/30'
    },
];

export const QuickAddMenu: React.FC<QuickAddMenuProps> = ({
                                                              x,
                                                              y,
                                                              onClose,
                                                              onSelect
                                                          }) => {
    const menuRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);
    const [filter, setFilter] = useState('');
    const [selectedIndex, setSelectedIndex] = useState(0);

    const filteredItems = ITEMS.filter(item =>
        item.label.toLowerCase().includes(filter.toLowerCase())
    );

    useEffect(() => {
        if (inputRef.current) inputRef.current.focus();

        const handleClickOutside = (event: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                onClose();
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, [onClose]);

    // Keyboard navigation
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'ArrowDown') {
                setSelectedIndex(i => (i + 1) % filteredItems.length);
                e.preventDefault();
            } else if (e.key === 'ArrowUp') {
                setSelectedIndex(i => (i - 1 + filteredItems.length) % filteredItems.length);
                e.preventDefault();
            } else if (e.key === 'Enter') {
                if (filteredItems[selectedIndex]) {
                    onSelect(filteredItems[selectedIndex].type);
                }
                e.preventDefault();
            } else if (e.key === 'Escape') {
                onClose();
            }
        };

        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [filteredItems, selectedIndex, onSelect, onClose]);


    return (
        <div
            ref={menuRef}
            className="fixed z-50 w-64 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-100 flex flex-col"
            style={{top: y, left: x}}
        >
            <div className="p-2 border-b border-slate-100 dark:border-slate-800 bg-slate-50 dark:bg-slate-900/50">
                <div className="relative">
                    <Search size={14} className="absolute left-2.5 top-2.5 text-slate-400"/>
                    <input
                        ref={inputRef}
                        type="text"
                        placeholder="Search nodes..."
                        value={filter}
                        onChange={(e) => {
                            setFilter(e.target.value);
                            setSelectedIndex(0);
                        }}
                        className="w-full pl-8 pr-3 py-1.5 text-sm bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg outline-none focus:border-blue-500 text-slate-800 dark:text-slate-200"
                    />
                </div>
            </div>
            <div className="max-h-[240px] overflow-y-auto py-1">
                {filteredItems.length === 0 ? (
                    <div className="px-4 py-3 text-xs text-slate-400 text-center">No nodes found</div>
                ) : (
                    filteredItems.map((item, index) => (
                        <button
                            key={item.type}
                            onClick={() => onSelect(item.type)}
                            className={`w-full text-left px-3 py-2 flex items-center transition-colors ${
                                index === selectedIndex
                                    ? 'bg-blue-50 dark:bg-blue-900/20'
                                    : 'hover:bg-slate-50 dark:hover:bg-slate-800'
                            }`}
                        >
                            <div className={`p-1.5 rounded-md mr-3 ${item.bg}`}>
                                <item.icon size={14} className={item.color}/>
                            </div>
                            <span className={`text-sm font-medium ${
                                index === selectedIndex ? 'text-blue-700 dark:text-blue-300' : 'text-slate-700 dark:text-slate-300'
                            }`}>
                        {item.label}
                    </span>
                        </button>
                    ))
                )}
            </div>
        </div>
    );
};
