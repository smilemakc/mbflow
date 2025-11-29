/**
 * Vue Router configuration
 */

import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: [
        {
            path: '/',
            redirect: '/workflows',
        },
        {
            path: '/workflows',
            name: 'workflows',
            component: () => import('@/views/WorkflowListView.vue'),
            meta: {
                title: 'Workflows',
            },
        },
        {
            path: '/workflows/:id',
            name: 'workflow-editor',
            component: () => import('@/views/WorkflowEditorView.vue'),
            meta: {
                title: 'Workflow Editor',
            },
        },
        {
            path: '/workflows/:id/executions',
            name: 'workflow-executions',
            component: () => import('@/views/ExecutionMonitorView.vue'),
            meta: {
                title: 'Executions',
            },
        },
        {
            path: '/executions',
            name: 'executions',
            component: () => import('@/views/ExecutionHistoryView.vue'),
            meta: {
                title: 'Execution History',
            },
        },
        {
            path: '/executions/:id',
            name: 'execution-detail',
            component: () => import('@/views/ExecutionMonitorView.vue'),
            meta: {
                title: 'Execution Detail',
            },
        },
    ],
})

// Navigation guard for page titles
router.beforeEach((to, _from, next) => {
    const title = to.meta.title as string
    if (title) {
        document.title = `${title} - MBFlow`
    }
    next()
})

export default router
