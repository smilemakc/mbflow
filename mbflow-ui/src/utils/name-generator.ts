/**
 * Simple random name generator for nodes.
 * Generates names in the format "adjective_noun".
 */

const ADJECTIVES = [
    'swift', 'calm', 'eager', 'jolly', 'kind', 'lively', 'brave', 'witty',
    'proud', 'gentle', 'happy', 'silly', 'zealous', 'clever', 'smart',
    'ancient', 'modern', 'future', 'cyber', 'digital', 'quantum', 'rapid',
    'steady', 'robust', 'agile', 'bright', 'cosmic', 'daring', 'epic'
];

const NOUNS = [
    'panda', 'tiger', 'lion', 'eagle', 'shark', 'whale', 'dolphin', 'falcon',
    'wolf', 'bear', 'fox', 'owl', 'hawk', 'raven', 'lynx',
    'router', 'server', 'proxy', 'agent', 'worker', 'runner', 'process',
    'beacon', 'pilot', 'scout', 'guide', 'guard', 'shield'
];

export function generateRandomName(): string {
    const adj = ADJECTIVES[Math.floor(Math.random() * ADJECTIVES.length)];
    const noun = NOUNS[Math.floor(Math.random() * NOUNS.length)];
    return `${adj}_${noun}`;
}
