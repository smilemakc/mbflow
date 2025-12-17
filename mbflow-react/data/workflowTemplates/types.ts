import React from 'react';
import {AppEdge, AppNode} from '@/types';
import {MarkerType} from 'reactflow';

export interface WorkflowTemplate {
    id: string;
    name: string;
    description: string;
    icon: React.ElementType;
    color: string;
    category: 'basic' | 'telegram' | 'ai' | 'data';
    nodes: AppNode[];
    edges: AppEdge[];
}

export const edge = (id: string, source: string, target: string, label?: string): AppEdge => ({
    id,
    source,
    target,
    type: 'smoothstep',
    animated: false,
    markerEnd: {type: MarkerType.ArrowClosed},
    ...(label ? {label, labelStyle: {fontSize: 10}} : {})
});
