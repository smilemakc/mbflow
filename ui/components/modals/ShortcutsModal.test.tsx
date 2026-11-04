import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ShortcutsModal } from './ShortcutsModal';
import { useUIStore } from '@/store/uiStore';

describe('ShortcutsModal', () => {
  beforeEach(() => useUIStore.setState({ language: 'en', activeModal: 'shortcuts' }));
  afterEach(() => vi.unstubAllGlobals());

  const keysFor = (description: string) =>
    Array.from(screen.getByText(description).parentElement!.querySelectorAll('kbd')).map((k) => k.textContent);

  it('lists every shortcut with its key combination', () => {
    vi.spyOn(navigator, 'platform', 'get').mockReturnValue('Linux x86_64');
    render(<ShortcutsModal />);
    expect(screen.getByRole('heading', { name: 'Keyboard Shortcuts' })).toBeInTheDocument();
    expect(keysFor('Save Workflow')).toEqual(['Ctrl', '+', 'S']);
    expect(keysFor('Toggle Focus Mode')).toEqual(['Shift', '+', 'F']);
    expect(keysFor('Delete Selected Node')).toEqual(['Delete']);
    expect(keysFor('Toggle Shortcuts')).toEqual(['Ctrl', '+', '/']);
    expect(screen.getAllByText(/^(Save Workflow|Run Workflow|Undo|Redo|Toggle Focus Mode|Delete Selected Node|Toggle Shortcuts)$/)).toHaveLength(7);
    expect(screen.getByText(/Pro tip/)).toBeInTheDocument();
  });

  it('shows the command symbol on macOS', () => {
    vi.spyOn(navigator, 'platform', 'get').mockReturnValue('MacIntel');
    render(<ShortcutsModal />);
    expect(keysFor('Undo')).toEqual(['⌘', '+', 'Z']);
  });

  it('closes by clearing the active modal', async () => {
    render(<ShortcutsModal />);
    await userEvent.click(screen.getByRole('button', { name: 'Close' }));
    expect(useUIStore.getState().activeModal).toBeNull();
  });
});
