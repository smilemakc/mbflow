import React, { useCallback, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { ReactFlowProvider } from 'reactflow';
import { useUIStore } from '@/store/uiStore';
import { useDagStore } from '@/store/dagStore';
import { Sidebar } from '@/components/navigation';
import { Header } from '@/components/builder';
import { NodeLibrary } from '@/components/builder/NodeLibrary';
import { DAGCanvas } from '@/components/builder/DAGCanvas';
import { PropertiesPanel } from '@/components/builder/PropertiesPanel';
import { MonitoringPanel } from '@/components/builder/MonitoringPanel';
import { EdgeConfigPanel } from '@/components/builder/EdgeConfigPanel';
import { ShortcutsModal } from '@/components/modals/ShortcutsModal';
import { TemplatesModal } from '@/components/modals/TemplatesModal';
import { WorkflowVariablesEditor } from '@/components/builder/WorkflowVariablesEditor';

export const BuilderLayout: React.FC = () => {
  const { isFullscreen, activeModal, setActiveModal, toggleFullscreen } = useUIStore();
  const { saveDAG, undo, redo, runWorkflow, fetchWorkflow, resetToNew } = useDagStore();
  const { workflowId } = useParams<{ workflowId: string }>();
  const navigate = useNavigate();

  // Load workflow from URL parameter or reset for new workflow
  useEffect(() => {
    if (workflowId) {
      fetchWorkflow(workflowId);
    } else {
      resetToNew();
    }
  }, [workflowId, fetchWorkflow, resetToNew]);

  // Save with redirect for new workflows
  const handleSave = useCallback(async () => {
    const newWorkflowId = await saveDAG();
    if (newWorkflowId) {
      // New workflow was created, redirect to its URL
      navigate(`/builder/${newWorkflowId}`, { replace: true });
    }
  }, [saveDAG, navigate]);

  // Global Keyboard Shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement).tagName)) {
        return;
      }

      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault();
        handleSave();
      } else if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        e.preventDefault();
        runWorkflow();
      } else if ((e.metaKey || e.ctrlKey) && e.key === 'z') {
        e.preventDefault();
        if (e.shiftKey) {
          redo();
        } else {
          undo();
        }
      } else if ((e.metaKey || e.ctrlKey) && e.key === 'y') {
        e.preventDefault();
        redo();
      } else if ((e.metaKey || e.ctrlKey) && e.key === '/') {
        e.preventDefault();
        setActiveModal(activeModal === 'shortcuts' ? null : 'shortcuts');
      } else if (e.shiftKey && e.key === 'F') {
        e.preventDefault();
        toggleFullscreen();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleSave, undo, redo, runWorkflow, activeModal, setActiveModal, toggleFullscreen]);

  return (
    <div className="flex h-screen w-screen bg-slate-50 dark:bg-slate-950 overflow-hidden font-sans text-slate-900 dark:text-slate-100 transition-colors duration-300">
      <ReactFlowProvider>
        {/* Modals */}
        {activeModal === 'shortcuts' && <ShortcutsModal />}
        {activeModal === 'templates' && <TemplatesModal />}
        <WorkflowVariablesEditor
          isOpen={activeModal === 'variables'}
          onClose={() => setActiveModal(null)}
        />

        {/* Sidebar - Hidden in Fullscreen */}
        {!isFullscreen && <Sidebar />}

        {/* Main Content Area */}
        <div className="flex-1 flex flex-col h-full min-w-0">
          <Header onSave={handleSave} />

          {/* Workspace */}
          <div className="flex-1 flex overflow-hidden relative">
            <div className="flex-1 relative h-full overflow-hidden flex flex-col">
              <NodeLibrary />

              <div className="flex-1 relative">
                <DAGCanvas />
                <MonitoringPanel />
              </div>

              <PropertiesPanel />
              <EdgeConfigPanel />
            </div>
          </div>
        </div>
      </ReactFlowProvider>
    </div>
  );
};
