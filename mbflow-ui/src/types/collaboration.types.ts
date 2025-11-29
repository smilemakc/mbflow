/**
 * Collaboration types - for real-time multi-user editing
 */

// ============================================================================
// User
// ============================================================================

export interface User {
    id: string
    name: string
    email?: string
    avatar?: string
    color: string // Hex color for cursor/selection
    initials: string
}

// ============================================================================
// Cursor Position
// ============================================================================

export interface CursorPosition {
    x: number
    y: number
    timestamp: number
}

// ============================================================================
// Resource Lock
// ============================================================================

export type ResourceType = 'node' | 'edge' | 'workflow'

export interface ResourceLock {
    resourceType: ResourceType
    resourceId: string
    userId: string
    acquiredAt: string
    expiresAt: string
}

// ============================================================================
// Client Events (sent to server)
// ============================================================================

export type ClientEvent =
    | {
        type: 'join_workflow'
        workflow_id: string
        user: User
    }
    | {
        type: 'leave_workflow'
        workflow_id: string
    }
    | {
        type: 'cursor_move'
        position: CursorPosition
    }
    | {
        type: 'node_update'
        node_id: string
        changes: Record<string, unknown>
    }
    | {
        type: 'edge_update'
        edge_id: string
        changes: Record<string, unknown>
    }
    | {
        type: 'lock_acquire'
        resource_type: ResourceType
        resource_id: string
    }
    | {
        type: 'lock_release'
        resource_type: ResourceType
        resource_id: string
    }

// ============================================================================
// Server Events (received from server)
// ============================================================================

export type ServerEvent =
    | {
        type: 'user_joined'
        user: User
    }
    | {
        type: 'user_left'
        user_id: string
    }
    | {
        type: 'cursor_update'
        user_id: string
        position: CursorPosition
    }
    | {
        type: 'node_updated'
        node_id: string
        changes: Record<string, unknown>
        user_id: string
    }
    | {
        type: 'edge_updated'
        edge_id: string
        changes: Record<string, unknown>
        user_id: string
    }
    | {
        type: 'lock_acquired'
        resource_type: ResourceType
        resource_id: string
        user_id: string
    }
    | {
        type: 'lock_released'
        resource_type: ResourceType
        resource_id: string
        user_id: string
    }
    | {
        type: 'conflict'
        resource_id: string
        conflicting_changes: Array<{
            user_id: string
            changes: Record<string, unknown>
        }>
    }

// ============================================================================
// Collaboration State
// ============================================================================

export interface CollaborationState {
    workflowId: string
    currentUser: User
    activeUsers: Map<string, User>
    userCursors: Map<string, CursorPosition>
    resourceLocks: Map<string, ResourceLock> // resourceId -> lock
    isConnected: boolean
    lastHeartbeat?: number
}

// ============================================================================
// Conflict Resolution
// ============================================================================

export interface Conflict {
    resourceId: string
    resourceType: ResourceType
    localChanges: Record<string, unknown>
    remoteChanges: Array<{
        userId: string
        userName: string
        changes: Record<string, unknown>
        timestamp: string
    }>
    resolvedAt?: string
}

export type ConflictResolutionStrategy = 'local' | 'remote' | 'merge' | 'manual'
