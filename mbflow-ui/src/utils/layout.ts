import ELK from 'elkjs/lib/elk.bundled.js'
import type { Node, Edge } from '@/types'

const elk = new ELK()

// ELK layout options
const defaultOptions = {
    'elk.algorithm': 'layered',
    'elk.direction': 'RIGHT',
    'elk.spacing.nodeNode': '80',
    'elk.layered.spacing.nodeNodeBetweenLayers': '100',
    'elk.padding': '[top=50,left=50,bottom=50,right=50]',
}

/**
 * Calculates the layout for the given nodes and edges using ELK.
 * Returns updated nodes with new positions.
 */
export async function getLayoutedElements(
    nodes: Node[],
    edges: Edge[],
    direction: 'RIGHT' | 'DOWN' = 'RIGHT'
): Promise<Node[]> {
    const isHorizontal = direction === 'RIGHT'

    const graph = {
        id: 'root',
        layoutOptions: {
            ...defaultOptions,
            'elk.direction': direction,
        },
        children: nodes.map((node) => ({
            id: node.id,
            width: 200, // Approximate width of a node
            height: 80, // Approximate height of a node
        })),
        edges: edges.map((edge) => ({
            id: edge.id,
            sources: [edge.from],
            targets: [edge.to],
        })),
    }

    try {
        const layoutedGraph = await elk.layout(graph)

        // Map back to our Node structure
        return nodes.map((node) => {
            const layoutedNode = layoutedGraph.children?.find((n) => n.id === node.id)

            if (layoutedNode) {
                return {
                    ...node,
                    metadata: {
                        ...node.metadata,
                        position: {
                            x: layoutedNode.x || 0,
                            y: layoutedNode.y || 0,
                        },
                    },
                }
            }

            return node
        })
    } catch (error) {
        console.error('ELK layout error:', error)
        return nodes
    }
}
