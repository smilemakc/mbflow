import React from 'react';
import {ArrowRight, Info} from 'lucide-react';

interface AdapterConfigProps {
    config: Record<string, any>;
    nodeId?: string;
    onChange: (config: Record<string, any>) => void;
}

const InfoBlock: React.FC<{ input: string; output: string }> = ({input, output}) => (
    <div
        className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-900/50 p-3 rounded-lg">
        <Info className="w-4 h-4 flex-shrink-0 text-blue-500"/>
        <div className="flex items-center gap-2 flex-1">
            <code className="bg-slate-200 dark:bg-slate-800 px-2 py-1 rounded font-mono text-[11px]">
                {input}
            </code>
            <ArrowRight className="w-3 h-3 flex-shrink-0"/>
            <code className="bg-slate-200 dark:bg-slate-800 px-2 py-1 rounded font-mono text-[11px]">
                {output}
            </code>
        </div>
    </div>
);

export const Base64ToBytesNodeConfig: React.FC<AdapterConfigProps> = () => {
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    Base64 to Bytes Converter
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    Decodes a base64-encoded string into raw bytes. Useful for processing binary data received as
                    base64.
                </p>
            </div>

            <InfoBlock
                input="SGVsbG8gV29ybGQ="
                output="b'Hello World'"
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                No configuration required. Input is automatically converted.
            </div>
        </div>
    );
};

export const BytesToBase64NodeConfig: React.FC<AdapterConfigProps> = () => {
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    Bytes to Base64 Converter
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    Encodes raw bytes into a base64 string. Useful for transmitting binary data as text.
                </p>
            </div>

            <InfoBlock
                input="b'Hello World'"
                output="SGVsbG8gV29ybGQ="
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                No configuration required. Input is automatically converted.
            </div>
        </div>
    );
};

export const StringToJsonNodeConfig: React.FC<AdapterConfigProps> = () => {
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    String to JSON Parser
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    Parses a JSON-formatted string into a structured object. Validates JSON syntax.
                </p>
            </div>

            <InfoBlock
                input='"{\"name\":\"Alice\"}"'
                output="{name: 'Alice'}"
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                No configuration required. Input is automatically parsed.
            </div>
        </div>
    );
};

export const JsonToStringNodeConfig: React.FC<AdapterConfigProps> = () => {
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    JSON to String Serializer
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    Serializes a structured object into a JSON-formatted string.
                </p>
            </div>

            <InfoBlock
                input="{name: 'Alice'}"
                output='"{\"name\":\"Alice\"}"'
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                No configuration required. Input is automatically serialized.
            </div>
        </div>
    );
};

export const BytesToJsonNodeConfig: React.FC<AdapterConfigProps> = () => {
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    Bytes to JSON Parser
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    Decodes bytes (UTF-8) and parses the result as JSON. Combines decoding and parsing in one step.
                </p>
            </div>

            <InfoBlock
                input={'b\'{"name":"Alice"}\''}
                output={'{"name": "Alice"}'}
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                No configuration required. Input is automatically decoded and parsed.
            </div>
        </div>
    );
};

export const FileToBytesNodeConfig: React.FC<AdapterConfigProps> = () => {
    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    File to Bytes Reader
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    Reads a file from storage and outputs its content as raw bytes.
                </p>
            </div>

            <InfoBlock
                input="File(id='abc123')"
                output="b'file content...'"
            />

            <div className="text-xs text-slate-500 dark:text-slate-500 italic">
                No configuration required. Accepts File object from storage nodes.
            </div>
        </div>
    );
};

export const BytesToFileNodeConfig: React.FC<AdapterConfigProps> = ({config, onChange}) => {
    const handleFilenameChange = (filename: string) => {
        onChange({...config, filename});
    };

    return (
        <div className="flex flex-col gap-4">
            <div>
                <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                    Bytes to File Writer
                </h3>
                <p className="text-xs text-slate-600 dark:text-slate-400 leading-relaxed">
                    Writes raw bytes to a file in storage. Specify the filename for the created file.
                </p>
            </div>

            <InfoBlock
                input="b'content...'"
                output="File(id='xyz789')"
            />

            <div className="flex flex-col gap-1.5">
                <label className="text-xs font-semibold text-slate-600 dark:text-slate-400">
                    Filename
                </label>
                <input
                    type="text"
                    value={config.filename || ''}
                    onChange={(e) => handleFilenameChange(e.target.value)}
                    placeholder="output.bin"
                    className="w-full px-3 py-2 text-sm bg-white dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 outline-none transition-all text-slate-800 dark:text-slate-200 placeholder-slate-400"
                />
                <p className="text-xs text-slate-500 dark:text-slate-500">
                    The name of the file to be created in storage
                </p>
            </div>
        </div>
    );
};
