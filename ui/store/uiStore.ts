import { create } from 'zustand';
import { persist } from 'zustand/middleware';

type Theme = 'light' | 'dark';
export type Language = 'ru' | 'en';
export type ModalType = 'shortcuts' | 'templates' | 'variables' | null;

interface UIState {
  theme: Theme;
  language: Language;
  isSidebarCollapsed: boolean;
  isFullscreen: boolean;
  activeModal: ModalType;
  isNodeLibraryOpen: boolean;
  
  toggleTheme: () => void;
  setTheme: (theme: Theme) => void;
  setLanguage: (lang: Language) => void;
  toggleLanguage: () => void;
  toggleSidebar: () => void;
  toggleFullscreen: () => void;
  setActiveModal: (modal: ModalType) => void;
  toggleNodeLibrary: () => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      theme: 'light',
      language: 'ru', // Default to Russian
      isSidebarCollapsed: false,
      isFullscreen: false,
      activeModal: null,
      isNodeLibraryOpen: false,

      toggleTheme: () => set((state) => ({ theme: state.theme === 'light' ? 'dark' : 'light' })),
      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
      toggleLanguage: () => set((state) => ({ language: state.language === 'ru' ? 'en' : 'ru' })),
      toggleSidebar: () => set((state) => ({ isSidebarCollapsed: !state.isSidebarCollapsed })),
      toggleFullscreen: () => set((state) => ({ isFullscreen: !state.isFullscreen })),
      setActiveModal: (modal) => set({ activeModal: modal }),
      toggleNodeLibrary: () => set((state) => ({ isNodeLibraryOpen: !state.isNodeLibraryOpen })),
    }),
    {
      name: 'ui-storage',
      partialize: (state) => ({ 
        theme: state.theme, 
        language: state.language,
        isSidebarCollapsed: state.isSidebarCollapsed,
        isNodeLibraryOpen: state.isNodeLibraryOpen
      }),
    }
  )
);