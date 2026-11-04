import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { UserSelector } from './UserSelector';
import { ADMIN_USERS } from '@/test/adminFixtures';

const toggle = () => screen.getAllByRole('button')[0];

describe('UserSelector', () => {
  it('shows the placeholder, label and required marker when nothing is selected', () => {
    render(<UserSelector value={null} onChange={vi.fn()} users={ADMIN_USERS} label="Owner" required />);
    expect(screen.getByText('Owner')).toBeInTheDocument();
    expect(screen.getByText('*')).toBeInTheDocument();
    expect(screen.getByText('Select user...')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Clear selection' })).not.toBeInTheDocument();
  });

  it('opens a searchable list and selects a user', async () => {
    const onChange = vi.fn();
    render(<UserSelector value={null} onChange={onChange} users={ADMIN_USERS} placeholder="Pick" />);
    await userEvent.click(screen.getByText('Pick'));

    // Initials from the full name, or the first letters of the username.
    expect(screen.getByText('AL')).toBeInTheDocument();
    expect(screen.getByText('BO')).toBeInTheDocument();
    expect(screen.getByText('@carol • carol@example.com')).toBeInTheDocument();

    const search = screen.getByPlaceholderText('Search users...');
    await userEvent.type(search, 'liddell');
    expect(screen.getByText('Alice Liddell')).toBeInTheDocument();
    expect(screen.queryByText('Bob')).not.toBeInTheDocument();

    await userEvent.clear(search);
    await userEvent.type(search, 'carol@');
    expect(screen.getByText('carol')).toBeInTheDocument();

    await userEvent.clear(search);
    await userEvent.type(search, 'nobody');
    expect(screen.getByText('No users found')).toBeInTheDocument();

    await userEvent.clear(search);
    await userEvent.click(screen.getByText('Bob'));
    expect(onChange).toHaveBeenCalledWith(ADMIN_USERS[1]);
    expect(screen.queryByPlaceholderText('Search users...')).not.toBeInTheDocument();
  });

  it('renders the selected user, highlights it and clears it without opening', async () => {
    const onChange = vi.fn();
    render(<UserSelector value={ADMIN_USERS[0]} onChange={onChange} users={ADMIN_USERS} />);
    expect(screen.getByText('Alice Liddell')).toBeInTheDocument();
    expect(screen.getByText('@alice')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'Clear selection' }));
    expect(onChange).toHaveBeenCalledWith(null);
    expect(screen.queryByPlaceholderText('Search users...')).not.toBeInTheDocument();

    fireEvent.keyDown(screen.getByRole('button', { name: 'Clear selection' }), { key: 'Enter' });
    expect(onChange).toHaveBeenCalledTimes(2);
    fireEvent.keyDown(screen.getByRole('button', { name: 'Clear selection' }), { key: 'a' });
    expect(onChange).toHaveBeenCalledTimes(2);

    await userEvent.click(toggle());
    const selectedRow = screen.getAllByText('Alice Liddell')[1].closest('button')!;
    expect(selectedRow.className).toContain('bg-blue-50');
  });

  it('uses username initials for a single-word full name and hides the clear button when asked', () => {
    render(
      <UserSelector value={ADMIN_USERS[1]} onChange={vi.fn()} users={ADMIN_USERS} showClearButton={false} />,
    );
    expect(screen.getByText('BO')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Clear selection' })).not.toBeInTheDocument();
  });

  it('shows a loading state', async () => {
    render(<UserSelector value={null} onChange={vi.fn()} users={[]} loading />);
    await userEvent.click(toggle());
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('does not open when disabled', async () => {
    render(<UserSelector value={ADMIN_USERS[0]} onChange={vi.fn()} users={ADMIN_USERS} disabled />);
    expect(toggle()).toBeDisabled();
    fireEvent.click(toggle());
    expect(screen.queryByPlaceholderText('Search users...')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Clear selection' })).not.toBeInTheDocument();
  });

  it('closes and resets the search on an outside click', async () => {
    render(
      <div>
        <p>outside</p>
        <UserSelector value={null} onChange={vi.fn()} users={ADMIN_USERS} />
      </div>,
    );
    await userEvent.click(toggle());
    await userEvent.type(screen.getByPlaceholderText('Search users...'), 'bob');
    fireEvent.mouseDown(screen.getByPlaceholderText('Search users...'));
    expect(screen.getByPlaceholderText('Search users...')).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByText('outside'));
    expect(screen.queryByPlaceholderText('Search users...')).not.toBeInTheDocument();

    await userEvent.click(toggle());
    expect(screen.getByPlaceholderText('Search users...')).toHaveValue('');
    await userEvent.click(toggle());
    expect(screen.queryByPlaceholderText('Search users...')).not.toBeInTheDocument();
  });
});
