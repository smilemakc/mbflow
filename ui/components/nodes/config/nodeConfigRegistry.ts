import { ComponentType } from 'react';
import { NodeType } from '@/types';

import { HTTPNodeConfigComponent } from './HTTPNodeConfig';
import { LLMNodeConfigComponent } from './LLMNodeConfig';
import { TelegramNodeConfigComponent } from './TelegramNodeConfig';
import { TelegramDownloadNodeConfig } from './TelegramDownloadNodeConfig';
import { TelegramParseNodeConfigComponent } from './TelegramParseNodeConfig';
import { TelegramCallbackNodeConfigComponent } from './TelegramCallbackNodeConfig';
import { ConditionalNodeConfigComponent } from './ConditionalNodeConfig';
import { MergeNodeConfigComponent } from './MergeNodeConfig';
import { DelayNodeConfig } from './DelayNodeConfig';
import { TransformNodeConfigComponent } from './TransformNodeConfig';
import { FunctionCallNodeConfigComponent } from './FunctionCallNodeConfig';
import { FileStorageNodeConfigComponent } from './FileStorageNodeConfig';
import {
    Base64ToBytesNodeConfig,
    BytesToBase64NodeConfig,
    BytesToJsonNodeConfig,
    BytesToFileNodeConfig,
    FileToBytesNodeConfig,
    JsonToStringNodeConfig,
    StringToJsonNodeConfig,
} from './AdapterNodeConfigs';
import { HTMLCleanNodeConfigComponent } from './HTMLCleanNodeConfig';
import { RSSParserNodeConfigComponent } from './RSSParserNodeConfig';
import { CSVToJSONNodeConfigComponent } from './CSVToJSONNodeConfig';
import { GoogleSheetsNodeConfigComponent } from './GoogleSheetsNodeConfig';
import { GoogleDriveNodeConfigComponent } from './GoogleDriveNodeConfig';
import { SubWorkflowNodeConfigComponent } from './SubWorkflowNodeConfig';

export interface NodeConfigProps<T = Record<string, any>> {
    config: T;
    nodeId?: string;
    onChange: (config: T) => void;
}

export interface NodeConfigEntry {
    component: ComponentType<NodeConfigProps>;
    label?: string;
    description?: string;
}

const registry: Map<string, NodeConfigEntry> = new Map();

export function registerNodeConfig(
    nodeType: NodeType | string,
    entry: NodeConfigEntry
): void {
    registry.set(nodeType, entry);
}

export function getNodeConfigComponent(
    nodeType: NodeType | string
): ComponentType<NodeConfigProps> | null {
    return registry.get(nodeType)?.component ?? null;
}

export function hasNodeConfig(nodeType: NodeType | string): boolean {
    return registry.has(nodeType);
}

export function getAllRegisteredNodeTypes(): string[] {
    return Array.from(registry.keys());
}

registerNodeConfig(NodeType.HTTP, {
    component: HTTPNodeConfigComponent,
    label: 'HTTP Request',
    description: 'Configure HTTP request parameters',
});

registerNodeConfig(NodeType.LLM, {
    component: LLMNodeConfigComponent,
    label: 'LLM / AI',
    description: 'Configure AI model parameters',
});

registerNodeConfig(NodeType.TELEGRAM, {
    component: TelegramNodeConfigComponent,
    label: 'Telegram Send',
    description: 'Send messages via Telegram',
});

registerNodeConfig(NodeType.TELEGRAM_DOWNLOAD, {
    component: TelegramDownloadNodeConfig,
    label: 'Telegram Download',
    description: 'Download files from Telegram',
});

registerNodeConfig(NodeType.TELEGRAM_PARSE, {
    component: TelegramParseNodeConfigComponent,
    label: 'Telegram Parse',
    description: 'Parse Telegram messages',
});

registerNodeConfig(NodeType.TELEGRAM_CALLBACK, {
    component: TelegramCallbackNodeConfigComponent,
    label: 'Telegram Callback',
    description: 'Handle Telegram callbacks',
});

registerNodeConfig(NodeType.CONDITIONAL, {
    component: ConditionalNodeConfigComponent,
    label: 'Conditional',
    description: 'Conditional branching logic',
});

registerNodeConfig(NodeType.MERGE, {
    component: MergeNodeConfigComponent,
    label: 'Merge',
    description: 'Merge multiple inputs',
});

registerNodeConfig(NodeType.DELAY, {
    component: DelayNodeConfig,
    label: 'Delay / Scheduler',
    description: 'Schedule or delay execution',
});

registerNodeConfig(NodeType.TRANSFORM, {
    component: TransformNodeConfigComponent,
    label: 'Transform',
    description: 'Transform data with JavaScript',
});

registerNodeConfig(NodeType.FUNCTION_CALL, {
    component: FunctionCallNodeConfigComponent,
    label: 'Function Call',
    description: 'Call custom functions',
});

registerNodeConfig(NodeType.FILE_STORAGE, {
    component: FileStorageNodeConfigComponent,
    label: 'File Storage',
    description: 'Store and retrieve files',
});

registerNodeConfig(NodeType.BASE64_TO_BYTES, {
    component: Base64ToBytesNodeConfig,
    label: 'Base64 to Bytes',
    description: 'Convert Base64 to byte array',
});

registerNodeConfig(NodeType.BYTES_TO_BASE64, {
    component: BytesToBase64NodeConfig,
    label: 'Bytes to Base64',
    description: 'Convert byte array to Base64',
});

registerNodeConfig(NodeType.STRING_TO_JSON, {
    component: StringToJsonNodeConfig,
    label: 'String to JSON',
    description: 'Parse JSON string',
});

registerNodeConfig(NodeType.JSON_TO_STRING, {
    component: JsonToStringNodeConfig,
    label: 'JSON to String',
    description: 'Stringify JSON object',
});

registerNodeConfig(NodeType.BYTES_TO_JSON, {
    component: BytesToJsonNodeConfig,
    label: 'Bytes to JSON',
    description: 'Convert bytes to JSON',
});

registerNodeConfig(NodeType.FILE_TO_BYTES, {
    component: FileToBytesNodeConfig,
    label: 'File to Bytes',
    description: 'Convert file to byte array',
});

registerNodeConfig(NodeType.BYTES_TO_FILE, {
    component: BytesToFileNodeConfig,
    label: 'Bytes to File',
    description: 'Convert byte array to file',
});

registerNodeConfig(NodeType.HTML_CLEAN, {
    component: HTMLCleanNodeConfigComponent,
    label: 'HTML Clean',
    description: 'Clean and sanitize HTML content',
});

registerNodeConfig(NodeType.RSS_PARSER, {
    component: RSSParserNodeConfigComponent,
    label: 'RSS Parser',
    description: 'Parse RSS feeds',
});

registerNodeConfig(NodeType.CSV_TO_JSON, {
    component: CSVToJSONNodeConfigComponent,
    label: 'CSV to JSON',
    description: 'Convert CSV to JSON format',
});

registerNodeConfig(NodeType.GOOGLE_SHEETS, {
    component: GoogleSheetsNodeConfigComponent,
    label: 'Google Sheets',
    description: 'Read/write Google Sheets data',
});

registerNodeConfig(NodeType.GOOGLE_DRIVE, {
    component: GoogleDriveNodeConfigComponent,
    label: 'Google Drive',
    description: 'Manage Google Drive files',
});

registerNodeConfig(NodeType.SUB_WORKFLOW, {
    component: SubWorkflowNodeConfigComponent,
    label: 'Sub-Workflow',
    description: 'Fan-out execution over an array',
});
