import React from 'react';
import {
  Database,
  Key,
  Globe,
  Plus,
  MoreVertical,
  Check,
  RefreshCw
} from 'lucide-react';
import { Button } from '@/components/ui';

export const ResourcesPage: React.FC = () => {
  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-5xl mx-auto space-y-8">
        
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Resources & Integrations</h1>
            <p className="text-slate-500 dark:text-slate-400 mt-1">Manage API keys, database connections, and external services.</p>
          </div>
          <Button variant="primary" icon={<Plus size={18} />}>
            Add Resource
          </Button>
        </div>

        {/* Categories */}
        <div className="space-y-6">
          
          {/* APIs */}
          <section>
            <h2 className="text-sm font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-4">API Connections</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <ResourceCard 
                title="OpenAI API" 
                type="API Key" 
                status="Active" 
                icon={<Key size={20} className="text-purple-500" />}
                meta="sk-proj...8421"
              />
              <ResourceCard 
                title="Stripe Production" 
                type="Payment Gateway" 
                status="Active" 
                icon={<Globe size={20} className="text-blue-500" />}
                meta="pk_live...9921"
              />
              <ResourceCard 
                title="SendGrid" 
                type="Email Service" 
                status="Error" 
                icon={<Globe size={20} className="text-orange-500" />}
                meta="Checking credentials..."
              />
            </div>
          </section>

          {/* Databases */}
          <section>
            <h2 className="text-sm font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-4">Databases</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <ResourceCard 
                title="Primary PostgreSQL" 
                type="Database" 
                status="Active" 
                icon={<Database size={20} className="text-slate-500" />}
                meta="postgres://app:***@db-prod:5432"
              />
              <ResourceCard 
                title="Redis Cache" 
                type="In-Memory Store" 
                status="Active" 
                icon={<Database size={20} className="text-red-500" />}
                meta="redis://cache-01:6379"
              />
            </div>
          </section>

        </div>
      </div>
    </div>
  );
};

const ResourceCard = ({ title, type, status, icon, meta }: any) => (
  <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-4 shadow-sm hover:border-blue-300 dark:hover:border-blue-700 transition-colors group">
    <div className="flex justify-between items-start">
      <div className="flex items-center space-x-3">
        <div className="p-2 bg-slate-50 dark:bg-slate-800 rounded-lg border border-slate-100 dark:border-slate-700 group-hover:bg-blue-50 dark:group-hover:bg-blue-900/20 transition-colors">
          {icon}
        </div>
        <div>
          <h3 className="font-bold text-slate-800 dark:text-slate-200">{title}</h3>
          <p className="text-xs text-slate-500 dark:text-slate-400">{type}</p>
        </div>
      </div>
      <Button variant="ghost" size="sm" icon={<MoreVertical size={16} />} />
    </div>
    
    <div className="mt-4 pt-4 border-t border-slate-100 dark:border-slate-800 flex justify-between items-center">
      <code className="text-xs font-mono text-slate-500 bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded">
        {meta}
      </code>
      <div className="flex items-center space-x-2">
         {status === 'Active' ? (
           <span className="flex items-center text-[10px] font-bold text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20 px-2 py-0.5 rounded-full">
             <Check size={10} className="mr-1" /> Active
           </span>
         ) : (
           <span className="flex items-center text-[10px] font-bold text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 px-2 py-0.5 rounded-full">
             <RefreshCw size={10} className="mr-1 animate-spin" /> Error
           </span>
         )}
      </div>
    </div>
  </div>
);
