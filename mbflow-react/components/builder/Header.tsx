import React, {useState, useRef, useEffect} from 'react';
import {useDagStore} from '@/store/dagStore';
import {useUIStore} from '@/store/uiStore';
import {useTranslation} from '@/store/translations';
import {useAutoSave} from '../../hooks/useAutoSave';
import {
    Check,
    ChevronRight,
    Home,
    Keyboard,
    LayoutTemplate,
    Loader2,
    Maximize,
    Minimize,
    Moon,
    Pencil,
    Play,
    Plus,
    RotateCcw,
    RotateCw,
    Save,
    Sun,
    Variable,
    X
} from 'lucide-react';
import {Button} from '../ui';

export const Header: React.FC = () => {
    const {
        isDirty,
        lastSavedAt,
        undo,
        redo,
        historyIndex,
        history,
        saveDAG,
        runWorkflow,
        isRunning,
        dagName,
        setDAGName
    } = useDagStore();

    const {
        theme,
        toggleTheme,
        isFullscreen,
        toggleFullscreen,
        setActiveModal,
        language,
        toggleLanguage,
        isNodeLibraryOpen,
        toggleNodeLibrary
    } = useUIStore();

    const t = useTranslation();

    const {isSaving} = useAutoSave();

    // Editable workflow name state
    const [isEditingName, setIsEditingName] = useState(false);
    const [editedName, setEditedName] = useState(dagName);
    const inputRef = useRef<HTMLInputElement>(null);

    // Sync editedName when dagName changes externally
    useEffect(() => {
        if (!isEditingName) {
            setEditedName(dagName);
        }
    }, [dagName, isEditingName]);

    // Focus input when editing starts
    useEffect(() => {
        if (isEditingName && inputRef.current) {
            inputRef.current.focus();
            inputRef.current.select();
        }
    }, [isEditingName]);

    const handleNameSave = () => {
        const trimmed = editedName.trim();
        if (trimmed && trimmed !== dagName) {
            setDAGName(trimmed);
        } else {
            setEditedName(dagName); // Reset to original if empty
        }
        setIsEditingName(false);
    };

    const handleNameCancel = () => {
        setEditedName(dagName);
        setIsEditingName(false);
    };

    const handleNameKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            handleNameSave();
        } else if (e.key === 'Escape') {
            handleNameCancel();
        }
    };

    return (
        <header
            className="h-16 bg-white dark:bg-slate-900 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between px-4 z-20 transition-colors shrink-0">

            {/* Left: Breadcrumbs & Status */}
            <div className="flex items-center min-w-0">
                <div className="flex items-center text-sm text-slate-500 dark:text-slate-400 mr-4">
                    <Home size={16} className="mr-2"/>
                    <span className="hidden sm:inline">{t.sidebar.workflows}</span>
                    <ChevronRight size={14} className="mx-2"/>

                    {/* Editable Workflow Name */}
                    {isEditingName ? (
                        <div className="flex items-center gap-1">
                            <input
                                ref={inputRef}
                                type="text"
                                value={editedName}
                                onChange={(e) => setEditedName(e.target.value)}
                                onKeyDown={handleNameKeyDown}
                                onBlur={handleNameSave}
                                className="px-2 py-1 text-sm font-semibold text-slate-900 dark:text-slate-100 bg-white dark:bg-slate-800 border border-blue-500 rounded-md outline-none min-w-[150px] max-w-[250px]"
                            />
                            <Button
                                variant="ghost"
                                size="sm"
                                icon={<Check size={14} />}
                                onClick={handleNameSave}
                                title="Save"
                                className="text-green-600 hover:bg-green-100 dark:hover:bg-green-900/30"
                            />
                            <Button
                                variant="ghost"
                                size="sm"
                                icon={<X size={14} />}
                                onClick={handleNameCancel}
                                title="Cancel"
                            />
                        </div>
                    ) : (
                        <button
                            onClick={() => setIsEditingName(true)}
                            className="group flex items-center gap-1.5 font-semibold text-slate-900 dark:text-slate-100 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                            title="Click to edit workflow name"
                        >
                            <span className="truncate max-w-[200px]">{dagName || 'Untitled Workflow'}</span>
                            <Pencil size={12} className="opacity-0 group-hover:opacity-100 transition-opacity"/>
                        </button>
                    )}
                </div>

                <div className="h-4 w-[1px] bg-slate-200 dark:bg-slate-700 mx-2 hidden sm:block"></div>

                {/* Components Toggle Button */}
                <Button
                    variant={isNodeLibraryOpen ? 'primary' : 'outline'}
                    size="sm"
                    icon={<Plus size={14} />}
                    onClick={toggleNodeLibrary}
                    className="hidden sm:flex ml-2"
                >
                    {t.builder.components}
                </Button>

                <div className="flex items-center space-x-2 ml-4">
          <span className={`text-xs px-2 py-0.5 rounded-full font-medium transition-colors ${
              isDirty
                  ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
                  : 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
          }`}>
            {isDirty ? t.common.unsaved : t.common.saved}
          </span>
                    {isSaving && <span className="text-xs text-slate-400 animate-pulse">{t.common.saving}</span>}
                </div>
            </div>

            {/* Right: Actions */}
            <div className="flex items-center space-x-2 sm:space-x-3">

                {/* Helper Group */}
                <div className="flex items-center gap-1">
                    <Button
                        variant="ghost"
                        icon={<Variable size={18} />}
                        onClick={() => setActiveModal('variables')}
                        title="Workflow Variables"
                    />
                    <Button
                        variant="ghost"
                        icon={<LayoutTemplate size={18} />}
                        onClick={() => setActiveModal('templates')}
                        title="Templates"
                    />
                    <Button
                        variant="ghost"
                        icon={<Keyboard size={18} />}
                        onClick={() => setActiveModal('shortcuts')}
                        title="Shortcuts"
                    />
                </div>

                <div className="h-6 w-[1px] bg-slate-200 dark:bg-slate-700 hidden sm:block"></div>

                {/* Undo/Redo Group */}
                <div
                    className="hidden sm:flex items-center bg-slate-100 dark:bg-slate-800 rounded-lg p-1 border border-slate-200 dark:border-slate-700">
                    <Button
                        variant="ghost"
                        size="sm"
                        icon={<RotateCcw size={16} />}
                        onClick={undo}
                        disabled={historyIndex === 0 || isRunning}
                        title={t.common.undo}
                    />
                    <Button
                        variant="ghost"
                        size="sm"
                        icon={<RotateCw size={16} />}
                        onClick={redo}
                        disabled={historyIndex === history.length - 1 || isRunning}
                        title={t.common.redo}
                    />
                </div>

                <div className="h-6 w-[1px] bg-slate-200 dark:bg-slate-700 hidden sm:block"></div>

                {/* View & Theme & Lang */}
                <div className="flex items-center gap-1">
                    <Button
                        variant="ghost"
                        onClick={toggleLanguage}
                        title="Switch Language"
                        className="font-bold text-xs w-9 h-9 border border-transparent hover:border-slate-200 dark:hover:border-slate-700"
                    >
                        {language.toUpperCase()}
                    </Button>

                    <Button
                        variant="ghost"
                        icon={isFullscreen ? <Minimize size={20}/> : <Maximize size={20}/>}
                        onClick={toggleFullscreen}
                        title="Focus Mode"
                        className={isFullscreen ? 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400' : ''}
                    />

                    <Button
                        variant="ghost"
                        icon={theme === 'dark' ? <Sun size={20}/> : <Moon size={20}/>}
                        onClick={toggleTheme}
                        title="Toggle Theme"
                    />
                </div>

                <Button
                    variant="outline"
                    icon={<Save size={16} />}
                    onClick={() => saveDAG()}
                    disabled={isRunning}
                    className="hidden sm:flex"
                >
                    {t.common.save}
                </Button>

                <Button
                    variant="primary"
                    icon={isRunning ? <Loader2 size={16} className="animate-spin"/> : <Play size={16} />}
                    onClick={() => runWorkflow()}
                    disabled={isRunning}
                    className={`shadow-lg ${isRunning ? 'shadow-none' : 'shadow-blue-500/20'} active:translate-y-0.5`}
                >
                    {isRunning ? t.common.running : t.common.run}
                </Button>
            </div>
        </header>
    );
};