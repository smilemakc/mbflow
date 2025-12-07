import React, {useCallback, useMemo, useRef, useState} from 'react';
import ReactFlow, {
    Background,
    Controls,
    MarkerType,
    MiniMap,
    OnConnectStartParams,
    ReactFlowInstance,
    useReactFlow
} from 'reactflow';
import {useDagStore} from '@/store/dagStore';
import {useUIStore} from '@/store/uiStore';
import {NodeType} from '@/types';
import {CustomNode} from './CustomNode';
import {ContextMenu} from './ContextMenu';
import {QuickAddMenu} from './QuickAddMenu';

// Define Custom Node Types
const nodeTypes = {
    custom: CustomNode,
};

// Main Canvas Component
export const DAGCanvas: React.FC = () => {
    const reactFlowWrapper = useRef<HTMLDivElement>(null);
    const [reactFlowInstance, setReactFlowInstance] = useState<ReactFlowInstance | null>(null);
    const {theme} = useUIStore();
    const {fitView} = useReactFlow();

    // Component State for interactions
    const [contextMenu, setContextMenu] = useState<{
        x: number;
        y: number;
        type: 'node' | 'pane';
        targetId?: string
    } | null>(null);
    const [quickAddMenu, setQuickAddMenu] = useState<{
        x: number;
        y: number;
        sourceNodeId: string;
        handleType: string | null
    } | null>(null);

    // Track where connection started
    const connectionStartRef = useRef<{ nodeId: string | null; handleType: string | null }>({
        nodeId: null,
        handleType: null
    });

    const {
        nodes,
        edges,
        onNodesChange,
        onEdgesChange,
        onConnect,
        addNode,
        setSelectedNodeId,
        setSelectedEdgeId,
        duplicateNode,
        deleteNode,
        updateNodeData
    } = useDagStore();

    const onDragOver = useCallback((event: React.DragEvent) => {
        event.preventDefault();
        event.dataTransfer.dropEffect = 'move';
    }, []);

    const onDrop = useCallback(
        (event: React.DragEvent) => {
            event.preventDefault();

            if (!reactFlowWrapper.current || !reactFlowInstance) return;

            const type = event.dataTransfer.getData('application/reactflow') as NodeType;
            if (!type) return;

            const position = reactFlowInstance.screenToFlowPosition({
                x: event.clientX,
                y: event.clientY,
            });

            addNode(type, position);
        },
        [reactFlowInstance, addNode]
    );

    const onNodeClick = useCallback((_: React.MouseEvent, node: any) => {
        setSelectedNodeId(node.id);
    }, [setSelectedNodeId]);

    const onEdgeClick = useCallback((_: React.MouseEvent, edge: any) => {
        setSelectedEdgeId(edge.id);
        setSelectedNodeId(null);
    }, [setSelectedEdgeId, setSelectedNodeId]);

    const onPaneClick = useCallback(() => {
        setSelectedNodeId(null);
        setSelectedEdgeId(null);
        setContextMenu(null);
        setQuickAddMenu(null);
    }, [setSelectedNodeId, setSelectedEdgeId]);

    // Context Menu Handlers
    const onNodeContextMenu = useCallback((event: React.MouseEvent, node: any) => {
        event.preventDefault();
        setContextMenu({
            x: event.clientX,
            y: event.clientY,
            type: 'node',
            targetId: node.id,
        });
    }, []);

    const onPaneContextMenu = useCallback((event: React.MouseEvent) => {
        event.preventDefault();
        setContextMenu({
            x: event.clientX,
            y: event.clientY,
            type: 'pane',
        });
    }, []);

    const handleContextMenuAction = (action: string, targetId?: string) => {
        if (action === 'duplicate' && targetId) {
            duplicateNode(targetId);
        } else if (action === 'delete' && targetId) {
            deleteNode(targetId);
        } else if (action === 'fit_view') {
            fitView({duration: 800});
        } else if (action === 'copy_id' && targetId) {
            navigator.clipboard.writeText(targetId);
        }
    };

    // Smart Connect Handlers
    const onConnectStart = useCallback((_: any, params: OnConnectStartParams) => {
        connectionStartRef.current = {nodeId: params.nodeId, handleType: params.handleType};
    }, []);

    const onConnectEnd = useCallback((event: MouseEvent | TouchEvent) => {
        const target = event.target as HTMLElement;
        // If we dropped on the pane (react-flow__pane), it means we didn't connect to another node
        const isPane = target.classList.contains('react-flow__pane');

        if (isPane && connectionStartRef.current.nodeId) {
            // Show quick add menu
            const {clientX, clientY} = event instanceof MouseEvent ? event : event.touches[0];
            setQuickAddMenu({
                x: clientX,
                y: clientY,
                sourceNodeId: connectionStartRef.current.nodeId,
                handleType: connectionStartRef.current.handleType
            });
        }
    }, []);

    const handleQuickAddSelect = useCallback((type: NodeType) => {
        if (!quickAddMenu || !reactFlowInstance) return;

        // Calculate position in flow
        const position = reactFlowInstance.screenToFlowPosition({
            x: quickAddMenu.x,
            y: quickAddMenu.y,
        });

        // 1. Add Node
        const newNode = addNode(type, position);

        // 2. Connect source to new node
        // If we dragged from Source Handle -> Connect to new node's Target
        // If we dragged from Target Handle -> Connect to new node's Source
        const source = quickAddMenu.handleType === 'source' ? quickAddMenu.sourceNodeId : newNode.id;
        const target = quickAddMenu.handleType === 'source' ? newNode.id : quickAddMenu.sourceNodeId;

        onConnect({
            source,
            target,
            sourceHandle: null,
            targetHandle: null
        });

        setQuickAddMenu(null);
    }, [quickAddMenu, reactFlowInstance, addNode, onConnect]);


    // Styles based on theme
    const bgColor = theme === 'dark' ? '#0f172a' : '#f8fafc'; // slate-950 : slate-50
    const dotColor = theme === 'dark' ? '#334155' : '#cbd5e1'; // slate-700 : slate-300
    const minimapMaskColor = theme === 'dark' ? 'rgba(15, 23, 42, 0.7)' : 'rgba(241, 245, 249, 0.7)';

    // Edge Styles
    const defaultEdgeOptions = useMemo(() => ({
        type: 'smoothstep',
        markerEnd: {type: MarkerType.ArrowClosed, color: theme === 'dark' ? '#94a3b8' : '#64748b'},
        style: {
            strokeWidth: 2,
            stroke: theme === 'dark' ? '#475569' : '#94a3b8',
        },
        animated: false,
    }), [theme]);

    const connectionLineStyle = useMemo(() => ({
        strokeWidth: 2,
        stroke: theme === 'dark' ? '#60a5fa' : '#3b82f6',
    }), [theme]);

    return (
        <div className="w-full h-full bg-slate-50 dark:bg-slate-950 transition-colors" ref={reactFlowWrapper}>
            <ReactFlow
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                onConnect={onConnect}
                onInit={setReactFlowInstance}
                onDrop={onDrop}
                onDragOver={onDragOver}
                onNodeClick={onNodeClick}
                onEdgeClick={onEdgeClick}
                onPaneClick={onPaneClick}
                onNodeContextMenu={onNodeContextMenu}
                onPaneContextMenu={onPaneContextMenu}
                onConnectStart={onConnectStart}
                onConnectEnd={onConnectEnd}
                nodeTypes={nodeTypes}
                defaultEdgeOptions={defaultEdgeOptions}
                connectionLineStyle={connectionLineStyle}
                fitView
                attributionPosition="bottom-right"
                style={{background: bgColor}}
                deleteKeyCode={['Backspace', 'Delete']}
            >
                <Background color={dotColor} gap={20} size={1}/>

                <Controls
                    className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 shadow-xl rounded-lg overflow-hidden [&>button]:bg-white dark:[&>button]:bg-slate-800 [&>button]:border-b-slate-200 dark:[&>button]:border-b-slate-700 [&>button:hover]:bg-slate-50 dark:[&>button:hover]:bg-slate-700 [&_svg]:!fill-slate-600 dark:[&_svg]:!fill-slate-100"/>

                <MiniMap
                    className="border border-slate-200 dark:border-slate-700 shadow-xl rounded-lg bg-white dark:bg-slate-800"
                    maskColor={minimapMaskColor}
                    nodeColor={() => theme === 'dark' ? '#475569' : '#94a3b8'}
                />
            </ReactFlow>

            {/* Context Menu */}
            {contextMenu && (
                <ContextMenu
                    x={contextMenu.x}
                    y={contextMenu.y}
                    type={contextMenu.type}
                    targetId={contextMenu.targetId}
                    onClose={() => setContextMenu(null)}
                    onAction={handleContextMenuAction}
                />
            )}

            {/* Quick Add Menu */}
            {quickAddMenu && (
                <QuickAddMenu
                    x={quickAddMenu.x}
                    y={quickAddMenu.y}
                    onClose={() => setQuickAddMenu(null)}
                    onSelect={handleQuickAddSelect}
                />
            )}
        </div>
    );
};