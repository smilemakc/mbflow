import { afterEach, describe, expect, it, vi } from 'vitest';
import React, { useState } from 'react';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { LLMApiTools } from './LLMApiTools';
import type { ApiToolRef } from '@/types/nodeConfigs';
import type { ApiMock } from '@/test/apiMock';
import { nodesEnT } from '@/test/nodesHarness';
import { nodesApiClient, nodesMockApiClients, nodesOperation } from '@/test/nodesApiClients';

const petstore = nodesApiClient('c1', [
  nodesOperation('listPets', { summary: 'List pets', tags: ['pets'] }),
  nodesOperation('createPet', { method: 'POST', path: '/pets', tags: ['pets'], description: 'Adds a pet' }),
  nodesOperation('ping', { path: '/ping', deprecated: true }),
]);
const weather = nodesApiClient('c2', [nodesOperation('forecast', { summary: 'Forecast' })]);

let api: ApiMock | undefined;
afterEach(() => api?.restore());

function setup(initial: ApiToolRef[] | undefined, otherTools = 0) {
  const onChange = vi.fn();
  function Harness() {
    const [value, setValue] = useState(initial);
    return (
      <LLMApiTools
        value={value}
        otherTools={otherTools}
        onChange={(v) => {
          onChange(v);
          setValue(v);
        }}
      />
    );
  }
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <Harness />
    </MemoryRouter>,
  );
  return { user, onChange };
}

describe('LLMApiTools', () => {
  const t = nodesEnT();
  const tt = t.nodeConfig.llm.apiTools;

  it('shows an import prompt when there are no API clients', async () => {
    api = nodesMockApiClients([]);
    setup(undefined);
    expect(await screen.findByText(tt.noClients)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: tt.importApi })).toHaveAttribute('href', '/api-clients');
    expect(screen.getByText(`${tt.title} (0)`)).toBeInTheDocument();
    expect(screen.queryByRole('combobox')).toBeNull();
  });

  it('adds a client, loads its operations and selects them', async () => {
    api = nodesMockApiClients([petstore, weather]);
    const { user, onChange } = setup(undefined);

    const picker = await screen.findByRole('combobox', { name: tt.addClient });
    expect(within(picker).getAllByRole('option').map((o) => o.textContent)).toEqual([tt.addClient, 'Client c1', 'Client c2']);
    await user.selectOptions(picker, 'c1');

    expect(await screen.findByText('List pets')).toBeInTheDocument();
    expect(api.calls.some((c) => c.url === '/api-clients/c1')).toBe(true);
    expect(screen.getByText('pets')).toBeInTheDocument();
    expect(screen.getByText(t.apiClients.untagged)).toBeInTheDocument();
    expect(screen.getByText('0 of 3 selected')).toBeInTheDocument();
    // c1 is no longer offered
    expect(within(screen.getByRole('combobox', { name: tt.addClient })).queryByRole('option', { name: 'Client c1' })).toBeNull();

    await user.click(screen.getByRole('checkbox', { name: /List pets/ }));
    expect(onChange).toHaveBeenLastCalledWith([{ client_id: 'c1', operation_ids: ['listPets'] }]);
    expect(screen.getByText(`${tt.title} (1)`)).toBeInTheDocument();

    await user.click(screen.getByRole('checkbox', { name: /POST \/pets/ }));
    expect(onChange).toHaveBeenLastCalledWith([{ client_id: 'c1', operation_ids: ['listPets', 'createPet'] }]);

    // Unchecking all keeps the client visible (pending) but clears the config.
    await user.click(screen.getByRole('button', { name: tt.clear }));
    expect(onChange).toHaveBeenLastCalledWith(undefined);
    expect(screen.getByText('0 of 3 selected')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: tt.clear })).toBeDisabled();
  });

  it('filters operations, selects the shown ones and removes the client', async () => {
    api = nodesMockApiClients([petstore, weather]);
    const { user, onChange } = setup([{ client_id: 'c2', operation_ids: ['forecast'] }]);

    // A client with a selection starts collapsed.
    const header = await screen.findByRole('button', { name: /Client c2/ });
    expect(header).toHaveAttribute('aria-expanded', 'false');
    await waitFor(() => expect(screen.getByText('1 of 1 selected')).toBeInTheDocument());

    await user.selectOptions(screen.getByRole('combobox', { name: tt.addClient }), 'c1');
    await screen.findByText('List pets');

    await user.type(screen.getByRole('searchbox', { name: tt.filter }), 'pets');
    expect(screen.queryByText('/ping')).toBeNull();
    await user.click(screen.getByRole('button', { name: tt.selectAll }));
    expect(onChange).toHaveBeenLastCalledWith([
      { client_id: 'c2', operation_ids: ['forecast'] },
      { client_id: 'c1', operation_ids: ['listPets', 'createPet'] },
    ]);

    await user.clear(screen.getByRole('searchbox', { name: tt.filter }));
    await user.type(screen.getByRole('searchbox', { name: tt.filter }), 'zzz');
    expect(screen.getByText(tt.noMatches)).toBeInTheDocument();

    await user.click(header);
    expect(header).toHaveAttribute('aria-expanded', 'true');

    const removeButtons = screen.getAllByRole('button', { name: tt.remove });
    await user.click(removeButtons[0]);
    expect(onChange).toHaveBeenLastCalledWith([{ client_id: 'c1', operation_ids: ['listPets', 'createPet'] }]);
    await user.click(screen.getByRole('button', { name: tt.remove }));
    expect(onChange).toHaveBeenLastCalledWith(undefined);
    expect(screen.queryByRole('button', { name: tt.remove })).toBeNull();
  });

  it('flags deleted clients, removed operations and too many tools', async () => {
    api = nodesMockApiClients([petstore]);
    const { user, onChange } = setup(
      [
        { client_id: 'gone', operation_ids: ['x'] },
        { client_id: 'c1', operation_ids: ['listPets', 'oldOp'] },
      ],
      127,
    );
    expect(await screen.findByText(tt.clientMissing.replace('{id}', 'gone'))).toBeInTheDocument();
    expect(screen.getByText(tt.tooMany.replace('{count}', '130').replace('{max}', '128'))).toBeInTheDocument();

    await user.click(await screen.findByRole('button', { name: /Client c1/ }));
    expect(screen.getByText('oldOp')).toBeInTheDocument();
    expect(screen.getByText(tt.operationMissing)).toBeInTheDocument();
    await user.click(screen.getByRole('checkbox', { name: /oldOp/ }));
    expect(onChange).toHaveBeenLastCalledWith([
      { client_id: 'gone', operation_ids: ['x'] },
      { client_id: 'c1', operation_ids: ['listPets'] },
    ]);
  });
});
