import React from 'react';
import {
    Activity,
    Boxes,
    ChevronLeft,
    ChevronRight,
    Database,
    FolderOpen,
    History,
    LayoutDashboard,
    LogOut,
    Settings,
    Workflow,
    Zap
} from 'lucide-react';
import {useLocation, useNavigate} from 'react-router-dom';
import {useUIStore} from '@/store/uiStore';
import {useTranslation} from '@/store/translations';
import {Button} from '../ui';

export const Sidebar: React.FC = () => {
    const {isSidebarCollapsed, toggleSidebar} = useUIStore();
    const location = useLocation();
    const navigate = useNavigate();
    const t = useTranslation();

    const isActive = (path: string) => {
        if (path === '/') return location.pathname === '/';
        return location.pathname.startsWith(path);
    };

    return (
        <div
            className={`${
                isSidebarCollapsed ? 'w-16' : 'w-64'
            } bg-slate-900 dark:bg-slate-950 text-slate-300 flex flex-col border-r border-slate-800 transition-all duration-300 z-30 flex-shrink-0 relative`}
        >
            {/* Logo Area */}
            <div
                className="h-16 flex items-center justify-center px-4 border-b border-slate-800 bg-slate-900 dark:bg-slate-950 whitespace-nowrap overflow-hidden">
                <div className="bg-blue-600 p-1.5 rounded-lg shadow-lg shadow-blue-900/20 flex-shrink-0 cursor-pointer"
                     onClick={() => navigate('/')}>
                    <Boxes className="w-5 h-5 text-white"/>
                </div>
                <span className={`ml-3 font-bold text-white text-lg tracking-tight transition-opacity duration-200 ${
                    isSidebarCollapsed ? 'opacity-0 w-0 hidden' : 'opacity-100'
                }`}>
          {t.sidebar.title}
        </span>
            </div>

            {/* Navigation */}
            <nav className="flex-1 py-6 space-y-1 px-3 overflow-y-auto overflow-x-hidden">
                <NavItem
                    icon={<LayoutDashboard size={20}/>}
                    label={t.sidebar.dashboard}
                    active={isActive('/dashboard')}
                    collapsed={isSidebarCollapsed}
                    onClick={() => navigate('/dashboard')}
                />
                <NavItem
                    icon={<FolderOpen size={20}/>}
                    label="Workflows List"
                    active={isActive('/workflows')}
                    collapsed={isSidebarCollapsed}
                    onClick={() => navigate('/workflows')}
                />
                <NavItem
                    icon={<Workflow size={20}/>}
                    label="Workflow Builder"
                    active={isActive('/builder')}
                    collapsed={isSidebarCollapsed}
                    onClick={() => navigate('/builder')}
                />
                <NavItem
                    icon={<History size={20}/>}
                    label={t.sidebar.executions}
                    active={isActive('/executions')}
                    collapsed={isSidebarCollapsed}
                    onClick={() => navigate('/executions')}
                />
                <NavItem
                    icon={<Zap size={20}/>}
                    label={t.sidebar.triggers}
                    active={isActive('/triggers')}
                    collapsed={isSidebarCollapsed}
                    onClick={() => navigate('/triggers')}
                />
                <NavItem
                    icon={<Activity size={20}/>}
                    label={t.sidebar.monitoring}
                    active={isActive('/monitoring')}
                    collapsed={isSidebarCollapsed}
                    onClick={() => navigate('/monitoring')}
                />
                <NavItem
                    icon={<Database size={20}/>}
                    label={t.sidebar.resources}
                    active={isActive('/resources')}
                    collapsed={isSidebarCollapsed}
                    onClick={() => navigate('/resources')}
                />
                <NavItem
                    icon={<Settings size={20}/>}
                    label={t.sidebar.settings}
                    active={isActive('/settings')}
                    collapsed={isSidebarCollapsed}
                    onClick={() => navigate('/settings')}
                />
            </nav>

            {/* Footer Actions */}
            <div className="p-4 border-t border-slate-800 space-y-2">
                <Button
                    variant="ghost"
                    size="sm"
                    fullWidth
                    icon={<LogOut size={20}/>}
                    className="text-slate-400 hover:text-white justify-center md:justify-start"
                >
                    <span className={`ml-3 text-sm font-medium transition-all duration-200 ${
                        isSidebarCollapsed ? 'opacity-0 w-0 hidden' : 'opacity-100'
                    }`}>
                        {t.sidebar.signOut}
                    </span>
                </Button>

                {/* Collapse Toggle */}
                <Button
                    variant="ghost"
                    size="sm"
                    fullWidth
                    icon={isSidebarCollapsed ? <ChevronRight size={16}/> : <ChevronLeft size={16}/>}
                    onClick={toggleSidebar}
                    className="text-slate-500 hover:text-white justify-center"
                />
            </div>
        </div>
    );
};

interface NavItemProps {
    icon: React.ReactNode;
    label: string;
    active?: boolean;
    collapsed?: boolean;
    onClick?: () => void;
}

const NavItem: React.FC<NavItemProps> = ({icon, label, active, collapsed, onClick}) => {
    return (
        <button
            onClick={onClick}
            className={`w-full flex items-center p-2.5 rounded-lg transition-all duration-200 group relative ${
                active
                    ? 'bg-blue-600 text-white shadow-md shadow-blue-900/20'
                    : 'hover:bg-slate-800 hover:text-white text-slate-400'
            } ${collapsed ? 'justify-center' : ''}`}
            title={collapsed ? label : undefined}
        >
            <span className={`ml-3 font-medium text-sm whitespace-nowrap transition-all duration-200 ${
                collapsed ? 'opacity-0 w-0 hidden' : 'opacity-100'
            }`}>
        {label}
      </span>
        </button>
    );
};
