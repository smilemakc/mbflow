import React from 'react';
import {Bell, CreditCard, Monitor, Save, Shield, User} from 'lucide-react';
import {useTranslation} from '@/store/translations';
import {Button} from '@/components/ui';

export const SettingsPage: React.FC = () => {
    const t = useTranslation();

    return (
        <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
            <div className="max-w-4xl mx-auto">
                <h1 className="text-2xl font-bold text-slate-900 dark:text-white mb-6">{t.settings.title}</h1>

                <div className="grid grid-cols-1 md:grid-cols-4 gap-6">

                    {/* Sidebar */}
                    <div className="md:col-span-1 space-y-1">
                        <NavButton icon={<User size={18}/>} label={t.settings.profile} active/>
                        <NavButton icon={<Bell size={18}/>} label={t.settings.notifications}/>
                        <NavButton icon={<Shield size={18}/>} label={t.settings.security}/>
                        <NavButton icon={<CreditCard size={18}/>} label={t.settings.billing}/>
                        <NavButton icon={<Monitor size={18}/>} label={t.settings.appearance}/>
                    </div>

                    {/* Content */}
                    <div className="md:col-span-3 space-y-6">

                        {/* Profile Section */}
                        <div
                            className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden">
                            <div className="p-6 border-b border-slate-100 dark:border-slate-800">
                                <h2 className="text-lg font-bold text-slate-900 dark:text-white">{t.settings.profileInfo}</h2>
                                <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{t.settings.profileDesc}</p>
                            </div>

                            <div className="p-6 space-y-4">
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-1.5">
                                        <label
                                            className="text-sm font-medium text-slate-700 dark:text-slate-300">{t.settings.firstName}</label>
                                        <input type="text" defaultValue="Alex"
                                               className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-slate-800 dark:text-slate-200"/>
                                    </div>
                                    <div className="space-y-1.5">
                                        <label
                                            className="text-sm font-medium text-slate-700 dark:text-slate-300">{t.settings.lastName}</label>
                                        <input type="text" defaultValue="Morgan"
                                               className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-slate-800 dark:text-slate-200"/>
                                    </div>
                                </div>

                                <div className="space-y-1.5">
                                    <label
                                        className="text-sm font-medium text-slate-700 dark:text-slate-300">{t.settings.email}</label>
                                    <input type="email" defaultValue="alex.morgan@example.com"
                                           className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-slate-800 dark:text-slate-200"/>
                                </div>

                                <div className="space-y-1.5">
                                    <label
                                        className="text-sm font-medium text-slate-700 dark:text-slate-300">{t.settings.bio}</label>
                                    <textarea rows={3} defaultValue="Senior Automation Engineer"
                                              className="w-full px-3 py-2 bg-white dark:bg-slate-950 border border-slate-300 dark:border-slate-700 rounded-lg focus:ring-2 focus:ring-blue-500 outline-none text-slate-800 dark:text-slate-200"/>
                                </div>
                            </div>

                            <div
                                className="p-4 bg-slate-50 dark:bg-slate-900/50 border-t border-slate-100 dark:border-slate-800 flex justify-end">
                                <Button variant="primary" size="sm" icon={<Save size={16}/>}>
                                    {t.settings.saveChanges}
                                </Button>
                            </div>
                        </div>

                        {/* Notification Preferences */}
                        <div
                            className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden">
                            <div className="p-6 border-b border-slate-100 dark:border-slate-800">
                                <h2 className="text-lg font-bold text-slate-900 dark:text-white">{t.settings.notifications}</h2>
                                <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">Choose what updates you
                                    want to receive.</p>
                            </div>
                            <div className="p-6 space-y-4">
                                <Toggle label="Workflow failures"
                                        description="Get notified when any production workflow fails." defaultChecked/>
                                <Toggle label="Weekly Digest" description="A summary of your automation performance."/>
                                <Toggle label="Security Alerts" description="Login attempts and password changes."
                                        defaultChecked/>
                            </div>
                        </div>

                    </div>
                </div>
            </div>
        </div>
    );
};

const NavButton = ({icon, label, active}: any) => (
    <button className={`w-full flex items-center px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
        active
            ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400'
            : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
    }`}>
        <span className="mr-3">{icon}</span>
        {label}
    </button>
);

const Toggle = ({label, description, defaultChecked}: any) => (
    <div className="flex items-start justify-between">
        <div>
            <div className="font-medium text-slate-800 dark:text-slate-200 text-sm">{label}</div>
            <div className="text-xs text-slate-500 dark:text-slate-400">{description}</div>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" className="sr-only peer" defaultChecked={defaultChecked}/>
            <div
                className="w-9 h-5 bg-slate-200 peer-focus:outline-none rounded-full peer dark:bg-slate-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all dark:border-slate-600 peer-checked:bg-blue-600"></div>
        </label>
    </div>
);