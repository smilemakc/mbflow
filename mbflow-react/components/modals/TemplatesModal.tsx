import React, {useState} from 'react';
import {
    ArrowRight,
    Bot,
    Database,
    FileText,
    Globe,
    Image,
    LayoutTemplate,
    MessageSquare,
    Share2,
    Webhook,
    X,
    Zap
} from 'lucide-react';
import {useUIStore} from '@/store/uiStore';
import {useDagStore} from '@/store/dagStore';
import {AppEdge, AppNode, NodeStatus, NodeType} from '@/types';
import {MarkerType} from 'reactflow';
import {Button, ConfirmModal} from '../ui';

// Template interface
interface WorkflowTemplate {
    id: string;
    name: string;
    description: string;
    icon: React.ElementType;
    color: string;
    category: 'basic' | 'telegram' | 'ai' | 'data';
    nodes: AppNode[];
    edges: AppEdge[];
}

// Helper to create edges
const edge = (id: string, source: string, target: string, label?: string): AppEdge => ({
    id,
    source,
    target,
    type: 'smoothstep',
    animated: false,
    markerEnd: {type: MarkerType.ArrowClosed},
    ...(label ? {label, labelStyle: {fontSize: 10}} : {})
});

// ============ TEMPLATES WITH FULL CONFIGURATIONS ============

const TEMPLATES: WorkflowTemplate[] = [
    // ===== BASIC TEMPLATES =====
    {
        id: 'api-data-transform',
        name: 'API Data Pipeline',
        description: 'Fetch data from REST API, transform it, and store or forward to another service.',
        icon: Globe,
        color: 'green',
        category: 'basic',
        nodes: [
            {
                id: 'http',
                type: 'custom',
                position: {x: 250, y: 50},
                data: {
                    label: 'Fetch Users API',
                    type: NodeType.HTTP,
                    status: NodeStatus.IDLE,
                    description: 'Fetch user data from external API',
                    config: {
                        url: 'https://jsonplaceholder.typicode.com/users',
                        method: 'GET',
                        headers: {
                            'Accept': 'application/json',
                            'User-Agent': 'MBFlow/1.0'
                        },
                        timeout_seconds: 30,
                        retry_count: 3,
                        follow_redirects: true
                    }
                }
            },
            {
                id: 'transform',
                type: 'custom',
                position: {x: 250, y: 200},
                data: {
                    label: 'Extract Names',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Transform API response to extract names and emails',
                    config: {
                        language: 'jq',
                        expression: '[.[] | {name: .name, email: .email, company: .company.name}]',
                        timeout_seconds: 10
                    }
                }
            },
            {
                id: 'http_2',
                type: 'custom',
                position: {x: 250, y: 350},
                data: {
                    label: 'Send to Webhook',
                    type: NodeType.HTTP,
                    status: NodeStatus.IDLE,
                    description: 'Forward transformed data to webhook endpoint',
                    config: {
                        url: '{{env.WEBHOOK_URL}}',
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer {{env.API_KEY}}'
                        },
                        body: '{"users": {{input.result}}}',
                        timeout_seconds: 30,
                        retry_count: 2
                    }
                }
            }
        ],
        edges: [
            edge('e1', 'http', 'transform'),
            edge('e2', 'transform', 'http_2')
        ]
    },

    {
        id: 'scheduled-health-check',
        name: 'Scheduled Health Monitor',
        description: 'Periodically check service health and alert on failures via conditional branching.',
        icon: Zap,
        color: 'orange',
        category: 'basic',
        nodes: [
            {
                id: 'delay',
                type: 'custom',
                position: {x: 250, y: 50},
                data: {
                    label: 'Every 5 Minutes',
                    type: NodeType.DELAY,
                    status: NodeStatus.IDLE,
                    description: 'Trigger health check every 5 minutes',
                    config: {
                        duration: 5,
                        unit: 'minutes',
                        description: 'Health check interval'
                    }
                }
            },
            {
                id: 'http',
                type: 'custom',
                position: {x: 250, y: 180},
                data: {
                    label: 'Health Check',
                    type: NodeType.HTTP,
                    status: NodeStatus.IDLE,
                    description: 'Check service health endpoint',
                    config: {
                        url: '{{env.SERVICE_URL}}/health',
                        method: 'GET',
                        headers: {},
                        timeout_seconds: 10,
                        retry_count: 1,
                        follow_redirects: false
                    }
                }
            },
            {
                id: 'conditional',
                type: 'custom',
                position: {x: 250, y: 310},
                data: {
                    label: 'Is Healthy?',
                    type: NodeType.CONDITIONAL,
                    status: NodeStatus.IDLE,
                    description: 'Check if service responded with 200 status',
                    config: {
                        condition: '{{input.status}} == 200 && {{input.body.status}} == "ok"'
                    }
                }
            },
            {
                id: 'transform',
                type: 'custom',
                position: {x: 100, y: 450},
                data: {
                    label: 'Log Success',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Format success log message',
                    config: {
                        language: 'jq',
                        expression: '{"status": "healthy", "timestamp": now | todate, "response_time_ms": .duration_ms}'
                    }
                }
            },
            {
                id: 'telegram',
                type: 'custom',
                position: {x: 400, y: 450},
                data: {
                    label: 'Alert Admin',
                    type: NodeType.TELEGRAM,
                    status: NodeStatus.IDLE,
                    description: 'Send alert to admin on failure',
                    config: {
                        bot_token: '{{env.TELEGRAM_BOT_TOKEN}}',
                        chat_id: '{{env.ADMIN_CHAT_ID}}',
                        message_type: 'text',
                        text: '🚨 <b>Service Down!</b>\n\nStatus: {{input.status}}\nURL: {{env.SERVICE_URL}}\nTime: {{input.timestamp}}',
                        parse_mode: 'HTML',
                        disable_notification: false
                    }
                }
            }
        ],
        edges: [
            edge('e1', 'delay', 'http'),
            edge('e2', 'http', 'conditional'),
            edge('e3', 'conditional', 'transform', 'true'),
            edge('e4', 'conditional', 'telegram', 'false')
        ]
    },

    // ===== TELEGRAM TEMPLATES =====
    {
        id: 'telegram-bot-handler',
        name: 'Telegram Bot Handler',
        description: 'Complete Telegram bot: receive messages, parse content, generate AI response, and reply.',
        icon: Bot,
        color: 'blue',
        category: 'telegram',
        nodes: [
            {
                id: 'telegram_parse',
                type: 'custom',
                position: {x: 250, y: 50},
                data: {
                    label: 'Parse Message',
                    type: NodeType.TELEGRAM_PARSE,
                    status: NodeStatus.IDLE,
                    description: 'Parse incoming Telegram update',
                    config: {
                        extract_files: true,
                        extract_commands: true,
                        extract_entities: true
                    }
                }
            },
            {
                id: 'conditional',
                type: 'custom',
                position: {x: 250, y: 180},
                data: {
                    label: 'Has Command?',
                    type: NodeType.CONDITIONAL,
                    status: NodeStatus.IDLE,
                    description: 'Check if message contains a command',
                    config: {
                        condition: 'len({{input.commands}}) > 0'
                    }
                }
            },
            {
                id: 'transform',
                type: 'custom',
                position: {x: 100, y: 310},
                data: {
                    label: 'Handle Command',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Process bot command',
                    config: {
                        language: 'jq',
                        expression: 'if .commands[0] == "/start" then "Welcome! I am your AI assistant. How can I help you today?" elif .commands[0] == "/help" then "Available commands:\n/start - Start conversation\n/help - Show this help" else "Unknown command. Try /help" end'
                    }
                }
            },
            {
                id: 'llm',
                type: 'custom',
                position: {x: 400, y: 310},
                data: {
                    label: 'AI Response',
                    type: NodeType.LLM,
                    status: NodeStatus.IDLE,
                    description: 'Generate response using LLM',
                    config: {
                        provider: 'openai',
                        model: 'gpt-4o-mini',
                        api_key: '{{env.OPENAI_API_KEY}}',
                        instruction: 'You are a helpful assistant in a Telegram chat. Be concise and friendly.',
                        prompt: '{{input.text}}',
                        temperature: 0.7,
                        max_tokens: 500,
                        response_format: 'text'
                    }
                }
            },
            {
                id: 'merge',
                type: 'custom',
                position: {x: 250, y: 450},
                data: {
                    label: 'Merge Response',
                    type: NodeType.MERGE,
                    status: NodeStatus.IDLE,
                    description: 'Merge command or AI response',
                    config: {
                        merge_strategy: 'first'
                    }
                }
            },
            {
                id: 'telegram',
                type: 'custom',
                position: {x: 250, y: 580},
                data: {
                    label: 'Send Reply',
                    type: NodeType.TELEGRAM,
                    status: NodeStatus.IDLE,
                    description: 'Send response back to user',
                    config: {
                        bot_token: '{{env.TELEGRAM_BOT_TOKEN}}',
                        chat_id: '{{input.chat.id}}',
                        message_type: 'text',
                        text: '{{input.merged.result || input.merged.content}}',
                        parse_mode: 'HTML',
                        disable_web_page_preview: true
                    }
                }
            }
        ],
        edges: [
            edge('e1', 'telegram_parse', 'conditional'),
            edge('e2', 'conditional', 'transform', 'true'),
            edge('e3', 'conditional', 'llm', 'false'),
            edge('e4', 'transform', 'merge'),
            edge('e5', 'llm', 'merge'),
            edge('e6', 'merge', 'telegram')
        ]
    },

    {
        id: 'telegram-file-processor',
        name: 'Telegram File Processor',
        description: 'Download files from Telegram, process them, and store in file storage.',
        icon: Image,
        color: 'cyan',
        category: 'telegram',
        nodes: [
            {
                id: 'telegram_parse',
                type: 'custom',
                position: {x: 250, y: 50},
                data: {
                    label: 'Parse Update',
                    type: NodeType.TELEGRAM_PARSE,
                    status: NodeStatus.IDLE,
                    description: 'Extract file info from Telegram message',
                    config: {
                        extract_files: true,
                        extract_commands: false,
                        extract_entities: false
                    }
                }
            },
            {
                id: 'conditional',
                type: 'custom',
                position: {x: 250, y: 180},
                data: {
                    label: 'Has File?',
                    type: NodeType.CONDITIONAL,
                    status: NodeStatus.IDLE,
                    description: 'Check if message contains a file',
                    config: {
                        condition: 'len({{input.files}}) > 0'
                    }
                }
            },
            {
                id: 'telegram_download',
                type: 'custom',
                position: {x: 250, y: 310},
                data: {
                    label: 'Download File',
                    type: NodeType.TELEGRAM_DOWNLOAD,
                    status: NodeStatus.IDLE,
                    description: 'Download file from Telegram servers',
                    config: {
                        bot_token: '{{env.TELEGRAM_BOT_TOKEN}}',
                        file_id: '{{input.files[0].file_id}}',
                        output_format: 'base64',
                        timeout: 120
                    }
                }
            },
            {
                id: 'base64_to_bytes',
                type: 'custom',
                position: {x: 250, y: 440},
                data: {
                    label: 'Decode Base64',
                    type: NodeType.BASE64_TO_BYTES,
                    status: NodeStatus.IDLE,
                    description: 'Convert base64 to bytes for storage',
                    config: {}
                }
            },
            {
                id: 'file_storage',
                type: 'custom',
                position: {x: 250, y: 570},
                data: {
                    label: 'Store File',
                    type: NodeType.FILE_STORAGE,
                    status: NodeStatus.IDLE,
                    description: 'Save file to storage',
                    config: {
                        action: 'store',
                        file_source: 'base64',
                        file_data: '{{input.bytes}}',
                        file_name: '{{input.files[0].file_name}}',
                        mime_type: '{{input.files[0].mime_type}}',
                        access_scope: 'workflow',
                        ttl: 86400,
                        tags: ['telegram', 'uploaded']
                    }
                }
            },
            {
                id: 'telegram',
                type: 'custom',
                position: {x: 250, y: 700},
                data: {
                    label: 'Confirm Upload',
                    type: NodeType.TELEGRAM,
                    status: NodeStatus.IDLE,
                    description: 'Send confirmation to user',
                    config: {
                        bot_token: '{{env.TELEGRAM_BOT_TOKEN}}',
                        chat_id: '{{input.chat.id}}',
                        message_type: 'text',
                        text: '✅ File saved!\n\n📄 Name: {{input.file_name}}\n📦 Size: {{input.size}} bytes\n🔗 ID: <code>{{input.file_id}}</code>',
                        parse_mode: 'HTML'
                    }
                }
            }
        ],
        edges: [
            edge('e1', 'telegram_parse', 'conditional'),
            edge('e2', 'conditional', 'telegram_download', 'true'),
            edge('e3', 'telegram_download', 'base64_to_bytes'),
            edge('e4', 'base64_to_bytes', 'file_storage'),
            edge('e5', 'file_storage', 'telegram')
        ]
    },

    // ===== AI TEMPLATES =====
    {
        id: 'content-generation-pipeline',
        name: 'AI Content Generator',
        description: 'Generate content using LLM, transform output, and publish to multiple channels.',
        icon: Share2,
        color: 'purple',
        category: 'ai',
        nodes: [
            {
                id: 'http',
                type: 'custom',
                position: {x: 250, y: 50},
                data: {
                    label: 'Get Topic',
                    type: NodeType.HTTP,
                    status: NodeStatus.IDLE,
                    description: 'Fetch trending topic or content brief',
                    config: {
                        url: '{{env.CONTENT_API}}/topics/trending',
                        method: 'GET',
                        headers: {
                            'Authorization': 'Bearer {{env.CONTENT_API_KEY}}'
                        },
                        timeout_seconds: 15
                    }
                }
            },
            {
                id: 'llm',
                type: 'custom',
                position: {x: 250, y: 200},
                data: {
                    label: 'Generate Article',
                    type: NodeType.LLM,
                    status: NodeStatus.IDLE,
                    description: 'Generate article content using GPT-4',
                    config: {
                        provider: 'openai',
                        model: 'gpt-4o',
                        api_key: '{{env.OPENAI_API_KEY}}',
                        instruction: 'You are a professional content writer. Write engaging, SEO-optimized content.',
                        prompt: 'Write a detailed article about: {{input.body.topic}}\n\nTarget audience: {{input.body.audience}}\nTone: Professional but accessible\nLength: 800-1000 words\n\nInclude:\n- Catchy headline\n- Introduction hook\n- 3-4 main sections with subheadings\n- Conclusion with call-to-action',
                        temperature: 0.8,
                        max_tokens: 2000,
                        response_format: 'text'
                    }
                }
            },
            {
                id: 'llm_2',
                type: 'custom',
                position: {x: 100, y: 380},
                data: {
                    label: 'Generate Summary',
                    type: NodeType.LLM,
                    status: NodeStatus.IDLE,
                    description: 'Create social media summary',
                    config: {
                        provider: 'openai',
                        model: 'gpt-4o-mini',
                        api_key: '{{env.OPENAI_API_KEY}}',
                        instruction: 'Create engaging social media posts.',
                        prompt: 'Create a compelling social media post (max 280 chars) summarizing this article:\n\n{{input.content}}',
                        temperature: 0.9,
                        max_tokens: 100
                    }
                }
            },
            {
                id: 'transform',
                type: 'custom',
                position: {x: 400, y: 380},
                data: {
                    label: 'Format for API',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Format content for CMS API',
                    config: {
                        language: 'jq',
                        expression: '{"title": (.content | split("\n")[0] | gsub("^#+ "; "")), "body": .content, "status": "draft", "author": "AI Writer", "tags": ["ai-generated", "trending"]}'
                    }
                }
            },
            {
                id: 'telegram',
                type: 'custom',
                position: {x: 100, y: 530},
                data: {
                    label: 'Post to Channel',
                    type: NodeType.TELEGRAM,
                    status: NodeStatus.IDLE,
                    description: 'Publish summary to Telegram channel',
                    config: {
                        bot_token: '{{env.TELEGRAM_BOT_TOKEN}}',
                        chat_id: '{{env.TELEGRAM_CHANNEL_ID}}',
                        message_type: 'text',
                        text: '📝 <b>New Article!</b>\n\n{{input.content}}\n\n🔗 Read more: {{env.WEBSITE_URL}}',
                        parse_mode: 'HTML'
                    }
                }
            },
            {
                id: 'http_2',
                type: 'custom',
                position: {x: 400, y: 530},
                data: {
                    label: 'Publish to CMS',
                    type: NodeType.HTTP,
                    status: NodeStatus.IDLE,
                    description: 'Save article to CMS',
                    config: {
                        url: '{{env.CMS_API}}/articles',
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer {{env.CMS_API_KEY}}'
                        },
                        body: '{{input.result | @json}}',
                        timeout_seconds: 30
                    }
                }
            }
        ],
        edges: [
            edge('e1', 'http', 'llm'),
            edge('e2', 'llm', 'llm_2'),
            edge('e3', 'llm', 'transform'),
            edge('e4', 'llm_2', 'telegram'),
            edge('e5', 'transform', 'http_2')
        ]
    },

    {
        id: 'ai-chatbot-with-tools',
        name: 'AI Agent with Tools',
        description: 'LLM-powered agent that can call external APIs and perform actions based on user requests.',
        icon: MessageSquare,
        color: 'violet',
        category: 'ai',
        nodes: [
            {
                id: 'transform',
                type: 'custom',
                position: {x: 250, y: 50},
                data: {
                    label: 'Parse Request',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Extract user message and context',
                    config: {
                        language: 'jq',
                        expression: '{"message": .text, "user_id": .from.id, "chat_id": .chat.id}'
                    }
                }
            },
            {
                id: 'llm',
                type: 'custom',
                position: {x: 250, y: 200},
                data: {
                    label: 'AI Decision',
                    type: NodeType.LLM,
                    status: NodeStatus.IDLE,
                    description: 'Analyze request and decide action',
                    config: {
                        provider: 'openai',
                        model: 'gpt-4o',
                        api_key: '{{env.OPENAI_API_KEY}}',
                        instruction: 'You are an AI assistant. Analyze the user request and respond with a JSON object indicating the action to take.\n\nAvailable actions:\n- "search": Search the web for information\n- "weather": Get weather information (requires: city)\n- "translate": Translate text (requires: text, target_language)\n- "respond": Just respond with text\n\nAlways respond with valid JSON: {"action": "...", "params": {...}, "response": "..."}',
                        prompt: 'User request: {{input.message}}',
                        temperature: 0.3,
                        max_tokens: 500,
                        response_format: 'json'
                    }
                }
            },
            {
                id: 'string_to_json',
                type: 'custom',
                position: {x: 250, y: 350},
                data: {
                    label: 'Parse JSON',
                    type: NodeType.STRING_TO_JSON,
                    status: NodeStatus.IDLE,
                    description: 'Parse LLM response as JSON',
                    config: {
                        strict_mode: true,
                        trim_whitespace: true
                    }
                }
            },
            {
                id: 'conditional',
                type: 'custom',
                position: {x: 250, y: 480},
                data: {
                    label: 'Route Action',
                    type: NodeType.CONDITIONAL,
                    status: NodeStatus.IDLE,
                    description: 'Route to appropriate handler',
                    config: {
                        condition: '{{input.json.action}} == "weather"'
                    }
                }
            },
            {
                id: 'http',
                type: 'custom',
                position: {x: 100, y: 620},
                data: {
                    label: 'Weather API',
                    type: NodeType.HTTP,
                    status: NodeStatus.IDLE,
                    description: 'Fetch weather data',
                    config: {
                        url: 'https://api.openweathermap.org/data/2.5/weather?q={{input.json.params.city}}&appid={{env.WEATHER_API_KEY}}&units=metric',
                        method: 'GET',
                        timeout_seconds: 10
                    }
                }
            },
            {
                id: 'transform_2',
                type: 'custom',
                position: {x: 400, y: 620},
                data: {
                    label: 'Format Response',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Format direct AI response',
                    config: {
                        language: 'jq',
                        expression: '.json.response'
                    }
                }
            },
            {
                id: 'merge',
                type: 'custom',
                position: {x: 250, y: 760},
                data: {
                    label: 'Merge Results',
                    type: NodeType.MERGE,
                    status: NodeStatus.IDLE,
                    description: 'Combine action results',
                    config: {
                        merge_strategy: 'first'
                    }
                }
            },
            {
                id: 'telegram',
                type: 'custom',
                position: {x: 250, y: 890},
                data: {
                    label: 'Send Response',
                    type: NodeType.TELEGRAM,
                    status: NodeStatus.IDLE,
                    description: 'Reply to user',
                    config: {
                        bot_token: '{{env.TELEGRAM_BOT_TOKEN}}',
                        chat_id: '{{input.chat_id}}',
                        message_type: 'text',
                        text: '{{input.merged}}',
                        parse_mode: 'HTML'
                    }
                }
            }
        ],
        edges: [
            edge('e1', 'transform', 'llm'),
            edge('e2', 'llm', 'string_to_json'),
            edge('e3', 'string_to_json', 'conditional'),
            edge('e4', 'conditional', 'http', 'true'),
            edge('e5', 'conditional', 'transform_2', 'false'),
            edge('e6', 'http', 'merge'),
            edge('e7', 'transform_2', 'merge'),
            edge('e8', 'merge', 'telegram')
        ]
    },

    // ===== DATA TEMPLATES =====
    {
        id: 'webhook-data-processor',
        name: 'Webhook Data Processor',
        description: 'Receive webhook data, validate, transform, and store with conditional error handling.',
        icon: Webhook,
        color: 'amber',
        category: 'data',
        nodes: [
            {
                id: 'transform',
                type: 'custom',
                position: {x: 250, y: 50},
                data: {
                    label: 'Validate Payload',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Validate incoming webhook payload',
                    config: {
                        language: 'jq',
                        expression: 'if .event_type and .data then {valid: true, event: .event_type, data: .data} else {valid: false, error: "Missing required fields"} end'
                    }
                }
            },
            {
                id: 'conditional',
                type: 'custom',
                position: {x: 250, y: 180},
                data: {
                    label: 'Is Valid?',
                    type: NodeType.CONDITIONAL,
                    status: NodeStatus.IDLE,
                    description: 'Check if payload is valid',
                    config: {
                        condition: '{{input.result.valid}} == true'
                    }
                }
            },
            {
                id: 'conditional_2',
                type: 'custom',
                position: {x: 150, y: 330},
                data: {
                    label: 'Event Type?',
                    type: NodeType.CONDITIONAL,
                    status: NodeStatus.IDLE,
                    description: 'Route based on event type',
                    config: {
                        condition: '{{input.result.event}} == "user.created"'
                    }
                }
            },
            {
                id: 'http',
                type: 'custom',
                position: {x: 450, y: 330},
                data: {
                    label: 'Log Error',
                    type: NodeType.HTTP,
                    status: NodeStatus.IDLE,
                    description: 'Log validation error',
                    config: {
                        url: '{{env.LOGGING_API}}/errors',
                        method: 'POST',
                        headers: {'Content-Type': 'application/json'},
                        body: '{"type": "webhook_validation_error", "error": "{{input.result.error}}", "timestamp": "{{input.timestamp}}"}'
                    }
                }
            },
            {
                id: 'transform_2',
                type: 'custom',
                position: {x: 50, y: 480},
                data: {
                    label: 'Process User',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Process user.created event',
                    config: {
                        language: 'jq',
                        expression: '.result.data | {user_id: .id, email: .email, name: .name, created_at: now | todate}'
                    }
                }
            },
            {
                id: 'transform_3',
                type: 'custom',
                position: {x: 250, y: 480},
                data: {
                    label: 'Process Other',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Process other events',
                    config: {
                        language: 'jq',
                        expression: '{event: .result.event, data: .result.data, processed_at: now | todate}'
                    }
                }
            },
            {
                id: 'merge',
                type: 'custom',
                position: {x: 150, y: 620},
                data: {
                    label: 'Merge Results',
                    type: NodeType.MERGE,
                    status: NodeStatus.IDLE,
                    description: 'Combine processed results',
                    config: {
                        merge_strategy: 'first'
                    }
                }
            },
            {
                id: 'http_2',
                type: 'custom',
                position: {x: 150, y: 750},
                data: {
                    label: 'Store to DB',
                    type: NodeType.HTTP,
                    status: NodeStatus.IDLE,
                    description: 'Store processed data',
                    config: {
                        url: '{{env.DATABASE_API}}/events',
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer {{env.DB_API_KEY}}'
                        },
                        body: '{{input.merged | @json}}',
                        timeout_seconds: 15,
                        retry_count: 3
                    }
                }
            }
        ],
        edges: [
            edge('e1', 'transform', 'conditional'),
            edge('e2', 'conditional', 'conditional_2', 'true'),
            edge('e3', 'conditional', 'http', 'false'),
            edge('e4', 'conditional_2', 'transform_2', 'true'),
            edge('e5', 'conditional_2', 'transform_3', 'false'),
            edge('e6', 'transform_2', 'merge'),
            edge('e7', 'transform_3', 'merge'),
            edge('e8', 'merge', 'http_2')
        ]
    },

    {
        id: 'file-etl-pipeline',
        name: 'File ETL Pipeline',
        description: 'Extract data from file, transform it, and load to multiple destinations.',
        icon: Database,
        color: 'teal',
        category: 'data',
        nodes: [
            {
                id: 'file_storage',
                type: 'custom',
                position: {x: 250, y: 50},
                data: {
                    label: 'Get Source File',
                    type: NodeType.FILE_STORAGE,
                    status: NodeStatus.IDLE,
                    description: 'Retrieve file from storage',
                    config: {
                        action: 'get',
                        file_id: '{{env.SOURCE_FILE_ID}}',
                        access_scope: 'workflow'
                    }
                }
            },
            {
                id: 'file_to_bytes',
                type: 'custom',
                position: {x: 250, y: 180},
                data: {
                    label: 'Read Content',
                    type: NodeType.FILE_TO_BYTES,
                    status: NodeStatus.IDLE,
                    description: 'Read file content as bytes',
                    config: {
                        file_id: '{{input.file_id}}',
                        output_format: 'base64'
                    }
                }
            },
            {
                id: 'bytes_to_json',
                type: 'custom',
                position: {x: 250, y: 310},
                data: {
                    label: 'Parse JSON',
                    type: NodeType.BYTES_TO_JSON,
                    status: NodeStatus.IDLE,
                    description: 'Parse file content as JSON',
                    config: {
                        encoding: 'utf-8',
                        validate_json: true
                    }
                }
            },
            {
                id: 'transform',
                type: 'custom',
                position: {x: 250, y: 440},
                data: {
                    label: 'Transform Data',
                    type: NodeType.TRANSFORM,
                    status: NodeStatus.IDLE,
                    description: 'Clean and transform data',
                    config: {
                        language: 'jq',
                        expression: '[.records[] | select(.status == "active") | {id: .id, name: (.first_name + " " + .last_name), email: .email | ascii_downcase, created: .created_at}]'
                    }
                }
            },
            {
                id: 'json_to_string',
                type: 'custom',
                position: {x: 100, y: 590},
                data: {
                    label: 'Format Output',
                    type: NodeType.JSON_TO_STRING,
                    status: NodeStatus.IDLE,
                    description: 'Convert to JSON string for storage',
                    config: {
                        pretty: true,
                        indent: '  ',
                        sort_keys: true
                    }
                }
            },
            {
                id: 'http',
                type: 'custom',
                position: {x: 400, y: 590},
                data: {
                    label: 'Send to API',
                    type: NodeType.HTTP,
                    status: NodeStatus.IDLE,
                    description: 'Send transformed data to destination API',
                    config: {
                        url: '{{env.DESTINATION_API}}/import',
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': 'Bearer {{env.DEST_API_KEY}}'
                        },
                        body: '{"records": {{input.result}}}',
                        timeout_seconds: 60,
                        retry_count: 3
                    }
                }
            },
            {
                id: 'bytes_to_file',
                type: 'custom',
                position: {x: 100, y: 730},
                data: {
                    label: 'Save Result',
                    type: NodeType.BYTES_TO_FILE,
                    status: NodeStatus.IDLE,
                    description: 'Save transformed data as new file',
                    config: {
                        file_name: 'transformed_{{input.timestamp}}.json',
                        mime_type: 'application/json',
                        access_scope: 'workflow',
                        ttl: 604800,
                        tags: ['etl', 'transformed', 'output']
                    }
                }
            }
        ],
        edges: [
            edge('e1', 'file_storage', 'file_to_bytes'),
            edge('e2', 'file_to_bytes', 'bytes_to_json'),
            edge('e3', 'bytes_to_json', 'transform'),
            edge('e4', 'transform', 'json_to_string'),
            edge('e5', 'transform', 'http'),
            edge('e6', 'json_to_string', 'bytes_to_file')
        ]
    }
];

// Category filter options
const CATEGORIES = [
    {id: 'all', label: 'All Templates'},
    {id: 'basic', label: 'Basic'},
    {id: 'telegram', label: 'Telegram'},
    {id: 'ai', label: 'AI & LLM'},
    {id: 'data', label: 'Data Processing'}
];

export const TemplatesModal: React.FC = () => {
    const {setActiveModal} = useUIStore();
    const {loadGraph} = useDagStore();
    const [selectedCategory, setSelectedCategory] = useState<string>('all');
    const [searchQuery, setSearchQuery] = useState('');
    const [templateToLoad, setTemplateToLoad] = useState<WorkflowTemplate | null>(null);

    const filteredTemplates = TEMPLATES.filter(t => {
        const matchesCategory = selectedCategory === 'all' || t.category === selectedCategory;
        const matchesSearch = searchQuery === '' ||
            t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            t.description.toLowerCase().includes(searchQuery.toLowerCase());
        return matchesCategory && matchesSearch;
    });

    const handleSelect = (template: WorkflowTemplate) => {
        setTemplateToLoad(template);
    };

    const handleConfirmLoad = () => {
        if (templateToLoad) {
            // Deep copy to ensure unique references
            const nodesCopy = JSON.parse(JSON.stringify(templateToLoad.nodes));
            const edgesCopy = JSON.parse(JSON.stringify(templateToLoad.edges));
            loadGraph(nodesCopy, edgesCopy);
            setActiveModal(null);
        }
        setTemplateToLoad(null);
    };

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm animate-in fade-in duration-200">
            <div
                className="w-full max-w-5xl bg-white dark:bg-slate-900 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-800 overflow-hidden transform animate-in zoom-in-95 duration-200 flex flex-col max-h-[90vh]">

                {/* Header */}
                <div
                    className="p-6 border-b border-slate-100 dark:border-slate-800 flex justify-between items-center bg-slate-50 dark:bg-slate-800/50">
                    <div>
                        <h2 className="text-xl font-bold text-slate-800 dark:text-slate-100 flex items-center">
                            <LayoutTemplate className="mr-3 text-blue-500" size={24}/>
                            Workflow Templates
                        </h2>
                        <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                            Start with a pre-built automation pattern with full configurations
                        </p>
                    </div>
                    <Button
                        onClick={() => setActiveModal(null)}
                        variant="ghost"
                        size="sm"
                        icon={<X size={20} />}
                    />
                </div>

                {/* Filters */}
                <div
                    className="px-6 py-4 border-b border-slate-100 dark:border-slate-800 flex flex-wrap gap-3 items-center bg-white dark:bg-slate-900">
                    {/* Category tabs */}
                    <div className="flex gap-2 flex-wrap">
                        {CATEGORIES.map(cat => (
                            <button
                                key={cat.id}
                                onClick={() => setSelectedCategory(cat.id)}
                                className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                                    selectedCategory === cat.id
                                        ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                                        : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
                                }`}
                            >
                                {cat.label}
                            </button>
                        ))}
                    </div>

                    {/* Search */}
                    <div className="flex-1 min-w-[200px]">
                        <input
                            type="text"
                            placeholder="Search templates..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="w-full px-3 py-1.5 text-sm bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 text-slate-700 dark:text-slate-200 placeholder-slate-400"
                        />
                    </div>
                </div>

                {/* Templates Grid */}
                <div className="flex-1 overflow-y-auto p-6 bg-slate-50/30 dark:bg-slate-950/30">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                        {filteredTemplates.map((t) => (
                            <div
                                key={t.id}
                                className="group bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl p-5 hover:border-blue-400 dark:hover:border-blue-600 hover:shadow-lg hover:shadow-blue-500/10 transition-all cursor-pointer flex flex-col"
                                onClick={() => handleSelect(t)}
                            >
                                <div className="flex justify-between items-start mb-3">
                                    <div
                                        className={`p-2.5 rounded-lg bg-${t.color}-50 dark:bg-${t.color}-900/20 text-${t.color}-600 dark:text-${t.color}-400 group-hover:scale-110 transition-transform`}>
                                        <t.icon size={22}/>
                                    </div>
                                    <div className="opacity-0 group-hover:opacity-100 transition-opacity">
                    <span
                        className="flex items-center text-xs font-bold text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30 px-2 py-1 rounded-full">
                      Use <ArrowRight size={12} className="ml-1"/>
                    </span>
                                    </div>
                                </div>

                                <h3 className="font-bold text-lg text-slate-900 dark:text-white mb-1.5">{t.name}</h3>
                                <p className="text-sm text-slate-500 dark:text-slate-400 mb-4 flex-1 leading-relaxed">{t.description}</p>

                                <div
                                    className="flex items-center justify-between pt-3 border-t border-slate-100 dark:border-slate-800">
                                    <div className="flex items-center space-x-3">
                    <span
                        className="text-xs font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded">
                      {t.nodes.length} nodes
                    </span>
                                        <span
                                            className="text-xs font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded">
                      {t.edges.length} edges
                    </span>
                                    </div>
                                    <span
                                        className={`text-xs font-medium px-2 py-0.5 rounded-full bg-${t.color}-50 dark:bg-${t.color}-900/20 text-${t.color}-600 dark:text-${t.color}-400`}>
                    {CATEGORIES.find(c => c.id === t.category)?.label}
                  </span>
                                </div>
                            </div>
                        ))}
                    </div>

                    {filteredTemplates.length === 0 && (
                        <div className="text-center py-12 text-slate-400">
                            <FileText size={48} className="mx-auto mb-3 opacity-50"/>
                            <p>No templates found matching your criteria</p>
                        </div>
                    )}
                </div>

            </div>

            {/* Confirm Load Template Modal */}
            <ConfirmModal
                isOpen={!!templateToLoad}
                onClose={() => setTemplateToLoad(null)}
                onConfirm={handleConfirmLoad}
                title="Load Template"
                message="This will replace your current workflow. Any unsaved changes will be lost. Continue?"
                confirmText="Load Template"
                variant="warning"
            />
        </div>
    );
};
