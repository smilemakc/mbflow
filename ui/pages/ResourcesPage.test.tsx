/**
 * ResourcesPage wires the resource hooks to its sections and modals. The
 * section components and data hooks are covered by their own tests, so they
 * are replaced by light stand-ins here to exercise the page's own wiring.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ResourcesPage } from './ResourcesPage';
import { useUIStore } from '@/store/uiStore';

const resources = vi.hoisted(() => ({
  state: {} as Record<string, any>,
  files: {} as Record<string, any>,
}));

vi.mock('@/hooks/resources/useResources.ts', () => ({ useResources: () => resources.state }));
vi.mock('@/hooks/resources/useFileStorage.ts', () => ({ useFileStorage: () => resources.files }));

vi.mock('@/components/resources', () => {
  const Modal = (name: string) => (props: any) =>
    props.isOpen ? (
      <div role="dialog" aria-label={name}>
        {name === 'Files' && <span>files of {props.resource?.name}: {props.files.length}</span>}
        {props.onSubmit && <button onClick={() => props.onSubmit({ amount: 10 })}>submit {name}</button>}
        <button onClick={props.onClose}>close {name}</button>
      </div>
    ) : null;
  return {
    AccountBalance: ({ account }: any) => <div>balance {account.balance}</div>,
    ResourceList: (props: any) => (
      <section>
        {props.resources.map((r: any) => (
          <div key={r.id}>
            {r.name}
            <button onClick={() => props.onViewFiles(r)}>view {r.name}</button>
            <button onClick={() => props.onDeleteResource(r.id, r.name)}>delete {r.name}</button>
          </div>
        ))}
        <button onClick={props.onCreateClick}>new storage</button>
      </section>
    ),
    CredentialList: () => <section>credentials</section>,
    RentalKeyList: () => <section>rental keys</section>,
    TransactionHistory: (props: any) => (
      <section>
        {props.total} transactions
        <button onClick={props.onDepositClick}>deposit</button>
      </section>
    ),
    CreateStorageModal: Modal('Create storage'),
    DepositModal: Modal('Deposit'),
    FilesModal: Modal('Files'),
  };
});

beforeEach(() => {
  useUIStore.setState({ language: 'en' });
  resources.state = {
    resources: [{ id: 'r1', name: 'Bucket A' }],
    account: { balance: 125 },
    transactions: [],
    transactionsTotal: 3,
    loading: false,
    loadData: vi.fn(),
    createStorage: vi.fn(),
    deleteResource: vi.fn().mockResolvedValue(undefined),
    deposit: vi.fn(),
  };
  resources.files = {
    selectedResource: null,
    files: [],
    filesTotal: 0,
    filesLoading: false,
    loadFiles: vi.fn(async (r) => {
      resources.files.selectedResource = r;
      resources.files.files = [{ id: 'f1' }, { id: 'f2' }];
    }),
    clearSelection: vi.fn(),
    uploadFile: vi.fn(),
    deleteFile: vi.fn(),
    downloadFile: vi.fn(),
  };
});

describe('ResourcesPage', () => {
  it('shows the header, balance and every section', () => {
    render(<ResourcesPage />);
    expect(screen.getByRole('heading', { level: 1 })).toBeInTheDocument();
    expect(screen.getByText('balance 125')).toBeInTheDocument();
    expect(screen.getByText('Bucket A')).toBeInTheDocument();
    expect(screen.getByText('credentials')).toBeInTheDocument();
    expect(screen.getByText('rental keys')).toBeInTheDocument();
    expect(screen.getByText('3 transactions')).toBeInTheDocument();
  });

  it('shows a spinner while loading and hides the balance without an account', () => {
    resources.state.loading = true;
    resources.state.account = null;
    render(<ResourcesPage />);
    expect(screen.queryByText('Bucket A')).toBeNull();
    expect(screen.queryByText(/^balance /)).toBeNull();
    expect(document.querySelector('.animate-spin')).not.toBeNull();
  });

  it('opens and closes the create-storage and deposit modals', async () => {
    const user = userEvent.setup();
    render(<ResourcesPage />);

    await user.click(screen.getByRole('button', { name: 'new storage' }));
    await user.click(screen.getByRole('button', { name: 'submit Create storage' }));
    expect(resources.state.createStorage).toHaveBeenCalledWith({ amount: 10 });
    await user.click(screen.getByRole('button', { name: 'close Create storage' }));
    expect(screen.queryByRole('dialog', { name: 'Create storage' })).toBeNull();

    await user.click(screen.getByRole('button', { name: 'deposit' }));
    await user.click(screen.getByRole('button', { name: 'submit Deposit' }));
    expect(resources.state.deposit).toHaveBeenCalledWith({ amount: 10 });
    await user.click(screen.getByRole('button', { name: 'close Deposit' }));
    expect(screen.queryByRole('dialog', { name: 'Deposit' })).toBeNull();
  });

  it('loads files before opening the files modal and clears the selection on close', async () => {
    const user = userEvent.setup();
    render(<ResourcesPage />);

    await user.click(screen.getByRole('button', { name: 'view Bucket A' }));
    expect(resources.files.loadFiles).toHaveBeenCalledWith({ id: 'r1', name: 'Bucket A' });
    expect(await screen.findByText('files of Bucket A: 2')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'close Files' }));
    expect(resources.files.clearSelection).toHaveBeenCalled();
    expect(screen.queryByRole('dialog', { name: 'Files' })).toBeNull();
  });

  it('deletes a resource only after confirmation', async () => {
    const user = userEvent.setup();
    render(<ResourcesPage />);

    await user.click(screen.getByRole('button', { name: 'delete Bucket A' }));
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(resources.state.deleteResource).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'delete Bucket A' }));
    await user.click(screen.getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(resources.state.deleteResource).toHaveBeenCalledWith('r1'));
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Cancel' })).toBeNull());
  });
});
