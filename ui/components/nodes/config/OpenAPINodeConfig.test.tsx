import { afterEach, describe, expect, it } from 'vitest';
import { fireEvent, screen, within } from '@testing-library/react';
import { OpenAPINodeConfigComponent } from './OpenAPINodeConfig';
import type { ApiMock } from '@/test/apiMock';
import { nodesEnT, nodesField, renderConfig } from '@/test/nodesHarness';
import { nodesApiClient, nodesMockApiClients, nodesOperation } from '@/test/nodesApiClients';

const pet = { $ref: '#/components/schemas/Pet' };
const client = nodesApiClient(
  'c1',
  [
    nodesOperation('getPet', {
      path: '/pets/{petId}',
      summary: 'Get a pet',
      tags: ['pets'],
      description: 'Returns one pet by id',
      parameters: [
        { name: 'petId', in: 'path', required: true, schema: { type: 'integer' }, example: 1, description: 'Pet id' },
        { name: 'verbose', in: 'query', schema: { type: 'boolean' } },
        { name: 'X-Trace', in: 'header', schema: { type: 'string', default: 'abc' } },
        { name: 'session', in: 'cookie', schema: { type: 'string', enum: ['s1', 's2'] } },
      ],
      responses: [{ status: '200', content: [{ content_type: 'application/json', schema: pet }] }],
    }),
    nodesOperation('createPet', {
      method: 'POST',
      path: '/pets',
      tags: ['pets'],
      deprecated: true,
      no_auth: true,
      parameters: [{ name: 'X-Trace', in: 'header', schema: { type: 'string' } }],
      request_body: {
        required: true,
        content: [
          { content_type: 'application/json', schema: pet },
          { content_type: 'application/xml', schema: { type: 'string' } },
        ],
      },
      responses: [{ status: 'default' }],
    }),
    nodesOperation('upload', {
      method: 'PUT',
      path: '/raw',
      request_body: { content: [{ content_type: 'text/plain', schema: { type: 'string' }, example: 'hello' }] },
    }),
  ],
  {
    auth: { type: 'bearer' },
    schemas: {
      Pet: {
        type: 'object',
        required: ['name'],
        properties: {
          id: { type: 'string', readOnly: true },
          age: { type: 'integer', description: 'Age in years' },
          name: { type: 'string', example: 'Rex' },
          tags: { type: 'array', items: { type: 'string' } },
          active: { type: 'boolean' },
          meta: { type: 'object', properties: { a: { type: 'string' } } },
        },
      },
    },
  },
);

let api: ApiMock | undefined;
afterEach(() => api?.restore());

const render = (config: Record<string, any>) =>
  renderConfig(OpenAPINodeConfigComponent, config, { router: true });

describe('OpenAPINodeConfig', () => {
  const t = nodesEnT();
  const o = t.nodeConfig.openapi;

  it('reports a node without a client and a deleted client', async () => {
    api = nodesMockApiClients([client]);
    const first = render({});
    expect(screen.getByText(o.clientMissing)).toBeInTheDocument();
    first.unmount();

    render({ client_id: 'deleted', operation_id: 'x' });
    expect(screen.getByText(o.loading)).toBeInTheDocument();
    expect(await screen.findByText(o.clientMissing)).toBeInTheDocument();
  });

  it('warns about a removed operation and selects another one, keeping shared settings', async () => {
    api = nodesMockApiClients([client]);
    const { user, last } = render({
      client_id: 'c1',
      operation_id: 'gone',
      base_url: 'https://override',
      path_params: { petId: '7', other: 'x' },
      headers: { 'x-trace': 't', 'X-Extra': 'e' },
      body: { name: 'Rex' },
      description: 'kept',
    });
    expect(await screen.findByText(o.operationMissing.replace('{id}', 'gone'))).toBeInTheDocument();
    const select = screen.getByRole('combobox', { name: o.operation });
    expect(within(select).getByRole('option', { name: 'gone' })).toBeInTheDocument();
    expect(within(select).getByRole('option', { name: 'GET Get a pet' })).toBeInTheDocument();
    expect(within(select).getByRole('option', { name: 'PUT /raw' })).toBeInTheDocument();

    await user.selectOptions(select, 'getPet');
    expect(last()).toEqual({
      client_id: 'c1',
      base_url: 'https://override',
      description: 'kept',
      operation_id: 'getPet',
      path_params: { petId: '7' },
      headers: { 'x-trace': 't' },
    });

    await user.selectOptions(select, 'createPet');
    expect(last()).toEqual({
      client_id: 'c1',
      base_url: 'https://override',
      description: 'kept',
      operation_id: 'createPet',
      headers: { 'x-trace': 't' },
    });
  });

  it('keeps the body when switching to another operation with a request body', async () => {
    api = nodesMockApiClients([client]);
    const { user, last } = render({ client_id: 'c1', operation_id: 'createPet', body: 'raw' });
    await user.selectOptions(await screen.findByRole('combobox', { name: o.operation }), 'upload');
    expect(last()).toEqual({ client_id: 'c1', operation_id: 'upload', body: 'raw' });
  });

  it('edits path, query, header and cookie parameters', async () => {
    api = nodesMockApiClients([client]);
    const { user, last } = render({ client_id: 'c1', operation_id: 'getPet', headers: { 'x-trace': 'old' } });

    expect(await screen.findByText('/pets/{petId}')).toBeInTheDocument();
    expect(screen.getByText('Authentication: Bearer token')).toBeInTheDocument();
    expect(screen.getByText(o.pathParams)).toBeInTheDocument();
    expect(screen.getByText(o.cookieParams)).toBeInTheDocument();
    expect(screen.getByText('Pet id')).toBeInTheDocument();
    expect(screen.getByText('Response 200: Pet')).toBeInTheDocument();

    const description = screen.getByRole('button', { name: 'Returns one pet by id' });
    expect(description.className).toContain('line-clamp-3');
    await user.click(description);
    expect(description.className).not.toContain('line-clamp-3');

    const petId = nodesField('petId');
    expect(petId).toHaveAttribute('placeholder', '1');
    await user.type(petId, '5');
    expect(last().path_params).toEqual({ petId: '5' });
    await user.clear(petId);
    expect(last()).not.toHaveProperty('path_params');

    const verboseSection = nodesField('verbose').closest('.space-y-1') as HTMLElement;
    await user.click(within(verboseSection).getByRole('button', { name: 'true' }));
    expect(last().query_params).toEqual({ verbose: 'true' });

    const trace = nodesField('X-Trace');
    expect(trace).toHaveValue('old');
    expect(trace).toHaveAttribute('placeholder', 'abc');
    fireEvent.change(trace, { target: { value: 'new' } });
    expect(last().headers).toEqual({ 'X-Trace': 'new' });

    const sessionSection = nodesField('session').closest('.space-y-1') as HTMLElement;
    await user.click(within(sessionSection).getByRole('button', { name: 's2' }));
    expect(last().cookies).toEqual({ session: 's2' });
  });

  it('edits the body field by field and converts it to typed raw JSON', async () => {
    api = nodesMockApiClients([client]);
    const { user, last } = render({ client_id: 'c1', operation_id: 'createPet' });

    expect(await screen.findByText(o.body)).toBeInTheDocument();
    expect(screen.getByText(t.apiClients.deprecated)).toBeInTheDocument();
    expect(screen.getByText(o.noAuth)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: o.fieldsMode })).toHaveAttribute('aria-pressed', 'true');
    // read-only id is not editable; required name comes first
    expect(screen.queryByText('id', { selector: 'label' })).toBeNull();
    const labels = Array.from(document.querySelectorAll('label.font-mono')).map((l) => l.textContent);
    expect(labels.slice(1)).toEqual(['name*', 'age', 'tags', 'active', 'meta']);
    expect(screen.getByText('Age in years')).toBeInTheDocument();

    await user.type(nodesField('name'), 'Rex');
    expect(last().body).toEqual({ name: 'Rex' });
    fireEvent.change(nodesField('age'), { target: { value: '3' } });
    fireEvent.change(nodesField('tags'), { target: { value: '["a"]' } });
    fireEvent.change(nodesField('meta'), { target: { value: '{bad' } });
    const activeSection = nodesField('active').closest('.space-y-1') as HTMLElement;
    await user.click(within(activeSection).getByRole('button', { name: 'true' }));
    expect(last().body).toEqual({ name: 'Rex', age: '3', tags: '["a"]', meta: '{bad', active: 'true' });
    fireEvent.change(nodesField('meta'), { target: { value: '' } });
    expect(last().body).not.toHaveProperty('meta');

    await user.click(screen.getByRole('button', { name: o.rawMode }));
    expect(JSON.parse(last().body)).toEqual({ name: 'Rex', age: 3, tags: ['a'], active: true });
    expect(screen.getByText(o.rawHint)).toBeInTheDocument();

    // switching back parses the JSON into text fields
    await user.click(screen.getByRole('button', { name: o.fieldsMode }));
    expect(last().body).toEqual({ name: 'Rex', age: '3', tags: '["a"]', active: 'true' });
  });

  it('refuses to show a non-object raw body as fields and inserts examples', async () => {
    api = nodesMockApiClients([client]);
    const { user, last } = render({ client_id: 'c1', operation_id: 'createPet', body: '[1, 2]' });
    await screen.findByText(o.body);
    expect(screen.getByRole('button', { name: o.rawMode })).toHaveAttribute('aria-pressed', 'true');
    await user.click(screen.getByRole('button', { name: o.rawMode }));
    await user.click(screen.getByRole('button', { name: o.fieldsMode }));
    expect(screen.getByText(o.bodyNotObject)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: o.insertExample }));
    expect(JSON.parse(last().body)).toEqual({ age: 0, name: 'Rex', tags: [''], active: false, meta: { a: '' } });

    // the example is an object, so the fields view opens now
    await user.click(screen.getByRole('button', { name: o.fieldsMode }));
    expect(screen.queryByText(o.bodyNotObject)).toBeNull();
    expect(nodesField('name')).toHaveValue('Rex');

    // an emptied raw body falls back to fields; insert example keeps typed values
    await user.click(screen.getByRole('button', { name: o.rawMode }));
    fireEvent.change(document.querySelector('textarea')!, { target: { value: '' } });
    expect(last()).not.toHaveProperty('body');
    expect(screen.getByRole('button', { name: o.fieldsMode })).toHaveAttribute('aria-pressed', 'true');
    await user.type(nodesField('name'), 'Bo');
    await user.click(screen.getByRole('button', { name: o.insertExample }));
    expect(last().body).toEqual({ age: '0', name: 'Bo', tags: '[""]', active: 'false', meta: '{"a":""}' });
  });

  it('switches content types and edits non-object bodies as text', async () => {
    api = nodesMockApiClients([client]);
    const { user, last } = render({ client_id: 'c1', operation_id: 'createPet' });
    const contentType = await screen.findByRole('combobox', { name: o.contentType });
    await user.selectOptions(contentType, 'application/xml');
    expect(last().content_type).toBe('application/xml');
    expect(screen.queryByRole('button', { name: o.fieldsMode })).toBeNull();
    fireEvent.change(document.querySelector('textarea')!, { target: { value: '<pet/>' } });
    expect(last().body).toBe('<pet/>');
    await user.selectOptions(contentType, 'application/json');
    expect(last()).not.toHaveProperty('content_type');
  });

  it('inserts a string example for a plain-text body', async () => {
    api = nodesMockApiClients([client]);
    const { user, last } = render({ client_id: 'c1', operation_id: 'upload' });
    await user.click(await screen.findByRole('button', { name: o.insertExample }));
    expect(last().body).toBe('hello');
  });

  it('switching an empty fields body to raw removes it', async () => {
    api = nodesMockApiClients([client]);
    const { user, last } = render({ client_id: 'c1', operation_id: 'createPet', body: {} });
    await screen.findByText(o.body);
    await user.click(screen.getByRole('button', { name: o.rawMode }));
    expect(last()).toEqual({ client_id: 'c1', operation_id: 'createPet' });
  });

  it('edits advanced settings', async () => {
    api = nodesMockApiClients([client]);
    const { user, last } = render({
      client_id: 'c1',
      operation_id: 'getPet',
      headers: { 'X-Trace': 't', 'X-Old': 'o' },
      query_params: { verbose: 'true' },
      response_type: 'binary',
      ignore_status_errors: true,
    });
    await user.click(await screen.findByText(o.advanced));
    expect(screen.getByText(o.baseUrlOverrideHint.replace('{url}', 'https://api.example.test'))).toBeInTheDocument();

    const baseUrl = screen.getByPlaceholderText('https://api.example.test');
    fireEvent.change(baseUrl, { target: { value: '  https://staging  ' } });
    expect(last().base_url).toBe('https://staging');

    // Only undeclared headers are listed as extra; declared ones are kept when editing.
    expect(screen.getByDisplayValue('X-Old')).toBeInTheDocument();
    expect(screen.queryByDisplayValue('X-Trace')).toBeNull();
    fireEvent.change(screen.getByDisplayValue('X-Old'), { target: { value: 'X-New' } });
    expect(last().headers).toEqual({ 'X-Trace': 't', 'X-New': 'o' });

    await user.click(screen.getByRole('button', { name: o.addParam }));
    const keyInputs = screen.getAllByPlaceholderText('param');
    await user.type(keyInputs[0], 'page');
    expect(last().query_params).toEqual({ verbose: 'true', page: '' });

    const responseType = nodesField<HTMLSelectElement>(o.responseType);
    expect(responseType).toHaveValue('binary');
    await user.selectOptions(responseType, 'auto');
    expect(last()).not.toHaveProperty('response_type');
    await user.selectOptions(responseType, 'binary');
    expect(last().response_type).toBe('binary');

    const ignore = nodesField(o.ignoreStatusErrors);
    expect(ignore).toBeChecked();
    await user.click(ignore);
    expect(last()).not.toHaveProperty('ignore_status_errors');
    await user.click(ignore);
    expect(last().ignore_status_errors).toBe(true);
  });
});
