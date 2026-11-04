import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { NodeType } from '@/types';
import {
  DefaultNodeConfig,
  getAllRegisteredNodeTypes,
  getNodeConfigComponent,
  hasNodeConfig,
  registerNodeConfig,
} from './index';
import { HTTPNodeConfigComponent } from './HTTPNodeConfig';
import { OpenAPINodeConfigComponent } from './OpenAPINodeConfig';
import { nodesEnT } from '@/test/nodesHarness';

describe('nodeConfigRegistry', () => {
  it('registers an editor for every built-in configurable node type', () => {
    const types = getAllRegisteredNodeTypes();
    expect(types).toEqual(
      expect.arrayContaining([
        NodeType.HTTP,
        NodeType.LLM,
        NodeType.TELEGRAM,
        NodeType.TELEGRAM_DOWNLOAD,
        NodeType.TELEGRAM_PARSE,
        NodeType.TELEGRAM_CALLBACK,
        NodeType.CONDITIONAL,
        NodeType.MERGE,
        NodeType.DELAY,
        NodeType.TRANSFORM,
        NodeType.FUNCTION_CALL,
        NodeType.FILE_STORAGE,
        NodeType.BASE64_TO_BYTES,
        NodeType.BYTES_TO_BASE64,
        NodeType.STRING_TO_JSON,
        NodeType.JSON_TO_STRING,
        NodeType.BYTES_TO_JSON,
        NodeType.FILE_TO_BYTES,
        NodeType.BYTES_TO_FILE,
        NodeType.HTML_CLEAN,
        NodeType.RSS_PARSER,
        NodeType.CSV_TO_JSON,
        NodeType.GOOGLE_SHEETS,
        NodeType.GOOGLE_DRIVE,
        NodeType.SUB_WORKFLOW,
        NodeType.OPENAPI,
      ]),
    );
    expect(getNodeConfigComponent(NodeType.HTTP)).toBe(HTTPNodeConfigComponent);
    expect(getNodeConfigComponent(NodeType.OPENAPI)).toBe(OpenAPINodeConfigComponent);
  });

  it('returns null / false for unknown types and allows registering custom editors', () => {
    expect(getNodeConfigComponent('no-such-type')).toBeNull();
    expect(hasNodeConfig('no-such-type')).toBe(false);

    const Custom = () => <div>custom editor</div>;
    registerNodeConfig('custom-test-type', { component: Custom, label: 'Custom' });
    expect(hasNodeConfig('custom-test-type')).toBe(true);
    expect(getNodeConfigComponent('custom-test-type')).toBe(Custom);
    expect(getAllRegisteredNodeTypes()).toContain('custom-test-type');
  });

  it('DefaultNodeConfig shows the no-configuration message', () => {
    const t = nodesEnT();
    render(<DefaultNodeConfig config={{}} onChange={() => {}} />);
    expect(screen.getByText(t.builder.noConfigAvailable || 'No specific configuration available for this node type.')).toBeInTheDocument();
  });
});
