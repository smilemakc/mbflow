/**
 * Page component types and interfaces
 */

import { ReactNode } from 'react';

/**
 * Base page component props
 */
export interface PageProps {
  className?: string;
  children?: ReactNode;
}

/**
 * Page types enumeration
 */
export enum PageType {
  DASHBOARD = 'dashboard',
  MONITORING = 'monitoring',
  SETTINGS = 'settings',
  RESOURCES = 'resources',
}

/**
 * Page metadata
 */
export interface PageMetadata {
  id: PageType;
  path: string;
  title: string;
  description: string;
  icon?: string;
}

/**
 * Available pages metadata
 */
export const PAGES_METADATA: Record<PageType, PageMetadata> = {
  [PageType.DASHBOARD]: {
    id: PageType.DASHBOARD,
    path: '/dashboard',
    title: 'Dashboard',
    description: 'Overview of your workflows and execution metrics',
  },
  [PageType.MONITORING]: {
    id: PageType.MONITORING,
    path: '/monitoring',
    title: 'Monitoring',
    description: 'Real-time infrastructure performance and error tracking',
  },
  [PageType.SETTINGS]: {
    id: PageType.SETTINGS,
    path: '/settings',
    title: 'Settings',
    description: 'User profile, notifications, and preferences',
  },
  [PageType.RESOURCES]: {
    id: PageType.RESOURCES,
    path: '/resources',
    title: 'Resources',
    description: 'Manage your resources and integrations',
  },
};
