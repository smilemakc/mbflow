import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  Base64ToBytesNodeConfig,
  BytesToBase64NodeConfig,
  BytesToFileNodeConfig,
  BytesToJsonNodeConfig,
  FileToBytesNodeConfig,
  JsonToStringNodeConfig,
  StringToJsonNodeConfig,
} from './AdapterNodeConfigs';
import { nodesEnT, renderConfig } from '@/test/nodesHarness';

describe('Adapter node configs', () => {
  const t = nodesEnT();
  const a = t.nodeConfig.adapter;

  it.each([
    ['base64ToBytes', Base64ToBytesNodeConfig, 'SGVsbG8gV29ybGQ=', "b'Hello World'"],
    ['bytesToBase64', BytesToBase64NodeConfig, "b'Hello World'", 'SGVsbG8gV29ybGQ='],
    ['stringToJson', StringToJsonNodeConfig, '"{\\"name\\":\\"Alice\\"}"', "{name: 'Alice'}"],
    ['jsonToString', JsonToStringNodeConfig, "{name: 'Alice'}", '"{\\"name\\":\\"Alice\\"}"'],
    ['bytesToJson', BytesToJsonNodeConfig, 'b\'{"name":"Alice"}\'', '{"name": "Alice"}'],
    ['fileToBytes', FileToBytesNodeConfig, "File(id='abc123')", "b'file content...'"],
  ] as const)('%s shows its title, conversion example and no-config notice', (key, Component, input, output) => {
    render(<Component config={{}} onChange={() => {}} />);
    const texts = a[key];
    expect(screen.getByRole('heading', { name: texts.title })).toBeInTheDocument();
    expect(screen.getByText(texts.description)).toBeInTheDocument();
    expect(screen.getByText(texts.noConfig)).toBeInTheDocument();
    expect(screen.getByText(input)).toBeInTheDocument();
    expect(screen.getByText(output)).toBeInTheDocument();
    expect(screen.queryByRole('textbox')).toBeNull();
  });

  it('BytesToFile edits the file name and keeps the other settings', async () => {
    const { user, last } = renderConfig(BytesToFileNodeConfig, { storage_id: 'default', filename: 'a' });
    const input = screen.getByPlaceholderText(a.bytesToFile.filenamePlaceholder);
    expect(input).toHaveValue('a');
    await user.type(input, 'b.txt');
    expect(last()).toEqual({ storage_id: 'default', filename: 'ab.txt' });
    expect(screen.getByText(a.bytesToFile.filenameHint)).toBeInTheDocument();
  });

  it('BytesToFile shows an empty field when no file name is set', () => {
    render(<BytesToFileNodeConfig config={{}} onChange={() => {}} />);
    expect(screen.getByPlaceholderText(a.bytesToFile.filenamePlaceholder)).toHaveValue('');
  });
});
