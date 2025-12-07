/**
 * Page utilities and helpers
 */

import { PageType, PAGES_METADATA } from '@/types/pages';

/**
 * Get page metadata by type
 */
export const getPageMetadata = (pageType: PageType) => {
  return PAGES_METADATA[pageType];
};

/**
 * Get page path by type
 */
export const getPagePath = (pageType: PageType): string => {
  return PAGES_METADATA[pageType]?.path || '/';
};

/**
 * Get page title by type
 */
export const getPageTitle = (pageType: PageType): string => {
  return PAGES_METADATA[pageType]?.title || 'Page';
};

/**
 * Get all available pages
 */
export const getAllPages = () => {
  return Object.values(PAGES_METADATA);
};

/**
 * Check if page exists
 */
export const isValidPageType = (pageType: any): pageType is PageType => {
  return Object.values(PageType).includes(pageType);
};

/**
 * Get page type from path
 */
export const getPageTypeFromPath = (path: string): PageType | null => {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;

  for (const [, metadata] of Object.entries(PAGES_METADATA)) {
    if (metadata.path === normalizedPath) {
      return metadata.id;
    }
  }

  return null;
};
