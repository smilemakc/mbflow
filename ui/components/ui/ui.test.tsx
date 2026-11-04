import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, within } from '@testing-library/react';
import { createRef } from 'react';
import { Inbox, Star } from 'lucide-react';
import {
  Button,
  ConfirmModal,
  DataTable,
  EmptyState,
  ErrorBanner,
  FieldGroup,
  FilterBar,
  FilterResults,
  FilterSearch,
  FilterSelect,
  InfoBox,
  LoadingState,
  Modal,
  Pagination,
  SectionHeader,
  StatusBadge,
  Toast,
  ToastContainer,
  type Column,
} from './index';
import { useToastStore } from '@/lib/toast';

const writeText = vi.fn();
beforeEach(() => {
  writeText.mockReset().mockResolvedValue(undefined);
  Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
});
afterEach(() => {
  vi.useRealTimers();
  useToastStore.getState().clearAll();
});

describe('Button', () => {
  it('renders variants, sizes and forwards refs and props', () => {
    const ref = createRef<HTMLButtonElement>();
    const onClick = vi.fn();
    render(
      <Button ref={ref} variant="danger" size="lg" fullWidth className="extra" onClick={onClick}>
        Delete
      </Button>,
    );
    const btn = screen.getByRole('button', { name: 'Delete' });
    expect(ref.current).toBe(btn);
    expect(btn.className).toContain('bg-red-600');
    expect(btn.className).toContain('px-6');
    expect(btn.className).toContain('w-full');
    expect(btn.className).toContain('extra');
    fireEvent.click(btn);
    expect(onClick).toHaveBeenCalled();
  });

  it('falls back to defaults for unknown variant and size', () => {
    render(<Button variant={'weird' as never} size={'xxl' as never}>X</Button>);
    const cls = screen.getByRole('button').className;
    expect(cls).toContain('bg-blue-600');
    expect(cls).toContain('px-4');
  });

  it('positions icons and disables while loading', () => {
    const { rerender } = render(
      <Button icon={<span data-testid="icon" />} variant="secondary">
        Left
      </Button>,
    );
    let btn = screen.getByRole('button');
    expect(btn.firstElementChild!.firstElementChild).toContainElement(screen.getByTestId('icon'));

    rerender(
      <Button icon={<span data-testid="icon" />} iconPosition="right" variant="ghost">
        Right
      </Button>,
    );
    btn = screen.getByRole('button');
    expect(btn.firstElementChild!.lastElementChild).toContainElement(screen.getByTestId('icon'));

    rerender(
      <Button loading variant="outline">
        Saving
      </Button>,
    );
    btn = screen.getByRole('button');
    expect(btn).toBeDisabled();
    expect(btn.querySelector('.animate-spin')).not.toBeNull();
  });
});

describe('simple presentational components', () => {
  it('EmptyState renders an optional action', () => {
    const onClick = vi.fn();
    const { rerender } = render(
      <EmptyState icon={Inbox} title="Nothing" description="Add one" action={{ label: 'Create', onClick, icon: <Star /> }} />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Create' }));
    expect(onClick).toHaveBeenCalled();
    rerender(<EmptyState icon={Inbox} title="Nothing" description="Add one" />);
    expect(screen.queryByRole('button')).toBeNull();
  });

  it('LoadingState shows an optional message', () => {
    const { rerender } = render(<LoadingState message="Loading..." />);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
    rerender(<LoadingState />);
    expect(screen.queryByText('Loading...')).toBeNull();
  });

  it('InfoBox supports variants, titles and custom icons', () => {
    const { container, rerender } = render(<InfoBox title="Heads up">Body</InfoBox>);
    expect(screen.getByText('Heads up')).toBeInTheDocument();
    expect(container.firstElementChild!.className).toContain('bg-blue-50');
    for (const [variant, cls] of [
      ['warning', 'bg-amber-50'],
      ['success', 'bg-green-50'],
      ['error', 'bg-red-50'],
    ] as const) {
      rerender(<InfoBox variant={variant} icon={Star}>Body</InfoBox>);
      expect(container.firstElementChild!.className).toContain(cls);
    }
    expect(screen.queryByText('Heads up')).toBeNull();
  });

  it('SectionHeader applies variant styles', () => {
    const { container, rerender } = render(<SectionHeader icon={Star} title="T" description="D" />);
    expect(container.firstElementChild!.className).toContain('from-blue-50');
    rerender(<SectionHeader icon={Star} title="T" description="D" variant="purple" />);
    expect(container.firstElementChild!.className).toContain('from-purple-50');
    expect(screen.getByText('D')).toBeInTheDocument();
  });

  it('StatusBadge accepts named or custom variants and icons', () => {
    const { container, rerender } = render(<StatusBadge status="OK" variant="success" size="sm" />);
    const badge = () => container.firstElementChild as HTMLElement;
    expect(badge().className).toContain('bg-green-50');
    expect(badge().className).toContain('px-2 ');
    rerender(<StatusBadge status="Custom" variant={{ bg: 'bg-x', text: 'text-x', border: 'border-x', icon: <i data-testid="vi" /> }} size="lg" />);
    expect(badge().className).toContain('bg-x');
    expect(screen.getByTestId('vi')).toBeInTheDocument();
    rerender(<StatusBadge status="Err" variant="error" icon={<i data-testid="own" />} />);
    expect(screen.getByTestId('own').parentElement!.className).toBe('w-3.5 h-3.5');
  });

  it('FieldGroup collapses only when collapsible', () => {
    const { rerender } = render(
      <FieldGroup title="Advanced" icon={<Star />}>
        <p>content</p>
      </FieldGroup>,
    );
    fireEvent.click(screen.getByText('Advanced'));
    expect(screen.getByText('content')).toBeInTheDocument();

    rerender(
      <FieldGroup key="collapsible" title="Advanced" collapsible defaultExpanded={false}>
        <p>content</p>
      </FieldGroup>,
    );
    expect(screen.queryByText('content')).toBeNull();
    fireEvent.click(screen.getByText('Advanced'));
    expect(screen.getByText('content')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Advanced'));
    expect(screen.queryByText('content')).toBeNull();
  });
});

describe('ErrorBanner', () => {
  it('copies the message and resets the icon', async () => {
    vi.useFakeTimers();
    render(<ErrorBanner title="Oops" message="stack trace" />);
    expect(screen.getByText('Oops')).toBeInTheDocument();
    const btn = screen.getByTitle('Copy to clipboard');
    const icon = () => btn.querySelector('svg')!.getAttribute('class');
    expect(icon()).toContain('text-red-600');
    await act(async () => {
      fireEvent.click(btn);
    });
    expect(writeText).toHaveBeenCalledWith('stack trace');
    expect(icon()).toContain('text-green-600');
    act(() => vi.advanceTimersByTime(2000));
    expect(icon()).toContain('text-red-600');
  });

  it('handles clipboard failures and can hide the copy button', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    writeText.mockRejectedValue(new Error('no'));
    const { rerender } = render(<ErrorBanner message="m" />);
    expect(screen.getByText('Error')).toBeInTheDocument();
    await act(async () => {
      fireEvent.click(screen.getByTitle('Copy to clipboard'));
    });
    expect(console.error).toHaveBeenCalled();
    rerender(<ErrorBanner message="m" showCopy={false} />);
    expect(screen.queryByTitle('Copy to clipboard')).toBeNull();
  });
});

describe('FilterBar', () => {
  it('composes select, search and result count', () => {
    const onSelect = vi.fn();
    const onSearch = vi.fn();
    const { rerender } = render(
      <FilterBar className="bar">
        <FilterSelect
          label="Status"
          value="all"
          onChange={onSelect}
          placeholder="Any"
          options={[
            { label: 'All', value: 'all' },
            { label: 'Active', value: 'active' },
          ]}
        />
        <FilterSearch value="" onChange={onSearch} />
        <FilterResults total={10} filtered={3} itemLabel="workflows" />
      </FilterBar>,
    );
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['Any', 'All', 'Active']);
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'active' } });
    expect(onSelect).toHaveBeenCalledWith('active');
    fireEvent.change(screen.getByPlaceholderText('Search...'), { target: { value: 'q' } });
    expect(onSearch).toHaveBeenCalledWith('q');
    expect(screen.getByText('3 of 10 workflows')).toBeInTheDocument();

    rerender(
      <FilterBar>
        <FilterSelect value="a" onChange={onSelect} options={[{ label: 'A', value: 'a' }]} />
        <FilterResults total={10} filtered={10} />
      </FilterBar>,
    );
    expect(screen.getByText('10 items')).toBeInTheDocument();
    expect(screen.getAllByRole('option')).toHaveLength(1);
  });
});

describe('Modal', () => {
  it('renders nothing when closed', () => {
    render(
      <Modal isOpen={false} onClose={() => {}}>
        body
      </Modal>,
    );
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(document.body.style.overflow).toBe('');
  });

  it('closes on escape, backdrop and close button and locks scrolling', () => {
    const onClose = vi.fn();
    const { rerender, unmount } = render(
      <Modal isOpen onClose={onClose} title="Title" footer={<span>footer</span>} size="xl">
        body
      </Modal>,
    );
    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-labelledby', 'modal-title');
    expect(dialog.className).toContain('max-w-xl');
    expect(screen.getByText('footer')).toBeInTheDocument();
    expect(document.body.style.overflow).toBe('hidden');

    fireEvent.keyDown(document, { key: 'Enter' });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByText('body'));
    expect(onClose).toHaveBeenCalledTimes(1);
    fireEvent.click(dialog.parentElement!);
    expect(onClose).toHaveBeenCalledTimes(2);

    fireEvent.click(screen.getByLabelText('Close modal'));
    expect(onClose).toHaveBeenCalledTimes(3);

    rerender(
      <Modal isOpen={false} onClose={onClose}>
        body
      </Modal>,
    );
    expect(document.body.style.overflow).toBe('');
    unmount();
  });

  it('respects disabled escape/backdrop and hides the header', () => {
    const onClose = vi.fn();
    render(
      <Modal isOpen onClose={onClose} closeOnEscape={false} closeOnBackdrop={false} showCloseButton={false}>
        body
      </Modal>,
    );
    const dialog = screen.getByRole('dialog');
    expect(dialog).not.toHaveAttribute('aria-labelledby');
    fireEvent.keyDown(document, { key: 'Escape' });
    fireEvent.click(dialog.parentElement!);
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.queryByLabelText('Close modal')).toBeNull();
  });
});

describe('ConfirmModal', () => {
  it('confirms and closes', () => {
    const onClose = vi.fn();
    const onConfirm = vi.fn();
    render(<ConfirmModal isOpen onClose={onClose} onConfirm={onConfirm} title="Delete?" message="Gone forever" confirmText="Yes" cancelText="No" />);
    expect(screen.getByText('Gone forever')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Yes' }).className).toContain('bg-red-600');
    fireEvent.click(screen.getByRole('button', { name: 'Yes' }));
    expect(onConfirm).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: 'No' }));
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it('supports other variants and blocks closing while loading', () => {
    const onClose = vi.fn();
    const { rerender } = render(<ConfirmModal isOpen onClose={onClose} onConfirm={() => {}} title="t" message="m" variant="warning" />);
    expect(screen.getByRole('button', { name: 'Confirm' }).className).toContain('bg-blue-600');
    rerender(<ConfirmModal isOpen onClose={onClose} onConfirm={() => {}} title="t" message="m" variant="info" isLoading />);
    expect(screen.getByRole('button', { name: 'Confirm' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
  });
});

describe('Toast', () => {
  const toast = { id: 't1', type: 'success' as const, title: 'Saved', message: 'All good', duration: 1000 };

  it('counts down the progress bar and closes', () => {
    vi.useFakeTimers();
    const onClose = vi.fn();
    const { container } = render(<Toast toast={toast} onClose={onClose} />);
    expect(screen.getByRole('alert')).toHaveTextContent('All good');
    const bar = () => (container.querySelector('[style]') as HTMLElement).style.width;
    expect(bar()).toBe('100%');
    act(() => vi.advanceTimersByTime(496));
    expect(parseFloat(bar())).toBeCloseTo(50, 0);
    act(() => vi.advanceTimersByTime(1000));
    expect(bar()).toBe('0%');
    expect(vi.getTimerCount()).toBe(0);
    fireEvent.click(screen.getByLabelText('Close notification'));
    expect(onClose).toHaveBeenCalledWith('t1');
  });

  it('omits the progress bar for sticky toasts', () => {
    const { container } = render(<Toast toast={{ ...toast, type: 'error', message: undefined, duration: 0 }} onClose={() => {}} />);
    expect(container.querySelector('[style]')).toBeNull();
    expect(screen.queryByText('All good')).toBeNull();
  });
});

describe('ToastContainer', () => {
  it('renders store toasts and dismisses them', () => {
    const { container } = render(<ToastContainer />);
    expect(container).toBeEmptyDOMElement();
    act(() => {
      useToastStore.getState().addToast({ type: 'info', title: 'A', duration: 0 });
      useToastStore.getState().addToast({ type: 'warning', title: 'B', duration: 0 });
    });
    expect(screen.getAllByRole('alert')).toHaveLength(2);
    fireEvent.click(within(screen.getAllByRole('alert')[0]).getByLabelText('Close notification'));
    expect(screen.getAllByRole('alert')).toHaveLength(1);
    expect(screen.getByText('B')).toBeInTheDocument();
  });
});

describe('Pagination', () => {
  it('warns and renders nothing without a mode', () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    const { container } = render(<Pagination />);
    expect(container).toBeEmptyDOMElement();
    expect(console.warn).toHaveBeenCalled();
  });

  it('navigates in offset mode with ellipses', () => {
    const onOffsetChange = vi.fn();
    const { rerender } = render(
      <Pagination offset={20} limit={10} total={95} onOffsetChange={onOffsetChange} showPageNumbers showItemCount itemLabel="runs" />,
    );
    expect(screen.getByText('Showing 21-30 of 95 runs')).toBeInTheDocument();
    expect(screen.getByLabelText('Go to page 3')).toHaveAttribute('aria-current', 'page');
    expect(screen.queryByLabelText('Go to page 6')).toBeNull();
    fireEvent.click(screen.getByLabelText('Previous page'));
    expect(onOffsetChange).toHaveBeenLastCalledWith(10);
    fireEvent.click(screen.getByLabelText('Next page'));
    expect(onOffsetChange).toHaveBeenLastCalledWith(30);
    fireEvent.click(screen.getByLabelText('Go to page 4'));
    expect(onOffsetChange).toHaveBeenLastCalledWith(30);
    fireEvent.click(screen.getByLabelText('Go to page 10'));
    expect(onOffsetChange).toHaveBeenLastCalledWith(90);

    rerender(<Pagination offset={90} limit={10} total={95} onOffsetChange={onOffsetChange} showPageNumbers size="md" />);
    expect(screen.getByLabelText('Next page')).toBeDisabled();
    expect(screen.getAllByLabelText(/Go to page/).map((b) => b.textContent)).toEqual(['1', '6', '7', '8', '9', '10']);
    fireEvent.click(screen.getByLabelText('Go to page 1'));
    expect(onOffsetChange).toHaveBeenLastCalledWith(0);
  });

  it('navigates in page mode', () => {
    const onPageChange = vi.fn();
    const { rerender } = render(
      <Pagination currentPage={2} totalPages={3} onPageChange={onPageChange} limit={10} total={25} showItemCount showPageNumbers size="lg" />,
    );
    expect(screen.getByText('Showing 11-20 of 25 items')).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('Previous page'));
    expect(onPageChange).toHaveBeenLastCalledWith(1);
    fireEvent.click(screen.getByLabelText('Next page'));
    expect(onPageChange).toHaveBeenLastCalledWith(3);
    fireEvent.click(screen.getByLabelText('Go to page 3'));
    expect(onPageChange).toHaveBeenLastCalledWith(3);

    rerender(<Pagination currentPage={1} totalPages={3} onPageChange={onPageChange} />);
    expect(screen.getByLabelText('Previous page')).toBeDisabled();
    expect(screen.queryByText(/Showing/)).toBeNull();
  });

  it('collapses to an item count or nothing for a single page', () => {
    const { container, rerender } = render(<Pagination offset={0} limit={10} total={4} onOffsetChange={() => {}} showItemCount />);
    expect(screen.getByText('Showing 4 items')).toBeInTheDocument();
    rerender(<Pagination offset={0} limit={10} total={4} onOffsetChange={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });
});

describe('DataTable', () => {
  interface Row {
    id: string;
    name: string;
    count?: number;
  }
  const rows: Row[] = [
    { id: '1', name: 'One', count: 3 },
    { id: '2', name: 'Two' },
  ];
  const columns: Column<Row>[] = [
    { key: 'name', header: 'Name', width: '50%' },
    { key: 'count', header: 'Count', align: 'right' },
    { key: 'idx', header: 'Idx', align: 'center', render: (r, i) => <b>{`${r.id}#${i}`}</b> },
  ];

  it('shows error, loading and empty states', () => {
    const onClick = vi.fn();
    const { rerender, container } = render(
      <DataTable data={[]} columns={columns} keyExtractor={(r) => r.id} error="Broken" />,
    );
    expect(screen.getByText('Broken')).toBeInTheDocument();

    rerender(<DataTable data={[]} columns={columns} keyExtractor={(r) => r.id} loading />);
    expect(container.querySelector('.animate-spin')).not.toBeNull();

    rerender(
      <DataTable
        data={[]}
        columns={columns}
        keyExtractor={(r) => r.id}
        emptyIcon={Inbox}
        emptyTitle="None"
        emptyAction={{ label: 'Add', onClick }}
      />,
    );
    expect(screen.getByText('None')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));
    expect(onClick).toHaveBeenCalled();

    rerender(<DataTable data={[]} columns={columns} keyExtractor={(r) => r.id} />);
    expect(screen.getByRole('table')).toBeInTheDocument();
    expect(screen.queryAllByRole('row')).toHaveLength(1);
  });

  it('renders cells, row clicks and actions', () => {
    const onRowClick = vi.fn();
    const onAction = vi.fn();
    render(
      <DataTable
        data={rows}
        columns={columns}
        keyExtractor={(r) => r.id}
        onRowClick={onRowClick}
        rowClassName={(r) => (r.id === '1' ? 'highlight' : '')}
        actions={(r) => <button onClick={() => onAction(r.id)}>Edit {r.id}</button>}
        actionsHeader="Ops"
        compact
        loading
      />,
    );
    expect(screen.getByText('Ops')).toBeInTheDocument();
    expect(screen.getByText('Name').getAttribute('style')).toContain('width: 50%');
    const [, first, second] = screen.getAllByRole('row');
    expect(first.className).toContain('highlight');
    expect(first.className).toContain('cursor-pointer');
    expect(within(first).getByText('3')).toBeInTheDocument();
    expect(within(first).getByText('1#0')).toBeInTheDocument();
    expect(within(second).getByText('2#1')).toBeInTheDocument();
    expect(within(second).getAllByRole('cell')[1].textContent).toBe('');

    fireEvent.click(within(second).getByText('Two'));
    expect(onRowClick).toHaveBeenCalledWith(rows[1]);
    fireEvent.click(screen.getByText('Edit 1'));
    expect(onAction).toHaveBeenCalledWith('1');
    expect(onRowClick).toHaveBeenCalledTimes(1);
  });

  it('pages through server data', () => {
    const onOffsetChange = vi.fn();
    const { rerender } = render(
      <DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} pagination={{ offset: 0, limit: 2, total: 5, onOffsetChange }} />,
    );
    expect(screen.getByText('Showing 1 - 2 of 5')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Next' }));
    expect(onOffsetChange).toHaveBeenLastCalledWith(2);

    rerender(
      <DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} pagination={{ offset: 4, limit: 2, total: 5, onOffsetChange }} />,
    );
    expect(screen.getByText('Showing 5 - 5 of 5')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Previous' }));
    expect(onOffsetChange).toHaveBeenLastCalledWith(2);

    rerender(
      <DataTable data={rows} columns={columns} keyExtractor={(r) => r.id} pagination={{ offset: 0, limit: 2, total: 0, onOffsetChange }} />,
    );
    expect(screen.queryByText(/Showing/)).toBeNull();
  });
});
