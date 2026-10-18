/** Drag data type of the node type string dropped onto the canvas. */
export const NODE_TYPE_MIME = 'application/reactflow';

/** Drag data type of a NodeTemplate that pre-fills the dropped node. */
export const NODE_TEMPLATE_MIME = 'application/x-mbflow-node-template';

export interface NodeTemplate {
    label?: string;
    description?: string;
    config?: Record<string, any>;
}

export const readNodeTemplate = (dataTransfer: DataTransfer): NodeTemplate | undefined => {
    const raw = dataTransfer.getData(NODE_TEMPLATE_MIME);
    if (!raw) return undefined;
    try {
        return JSON.parse(raw) as NodeTemplate;
    } catch {
        return undefined;
    }
};
