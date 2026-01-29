/**
 * Example: Using Eyesite SDK with Anthropic
 */

import { Eyesite } from '@eyesite/sdk';
import Anthropic from '@anthropic-ai/sdk';

async function main() {
    // Initialize Eyesite
    const eyesite = new Eyesite({
        apiKey: process.env.EYESITE_API_KEY || 'your-api-key',
        endpoint: 'http://localhost:8001',
        workspaceUuid: 'your-workspace-uuid',
    });

    // Create a session
    const sessionId = await eyesite.createSession('Anthropic Example');
    console.log('Created session:', sessionId);

    // Initialize Anthropic
    const anthropic = new Anthropic({
        apiKey: process.env.ANTHROPIC_API_KEY,
    });

    // Wrap Anthropic for automatic tracking
    eyesite.wrapAnthropic(anthropic);

    // Make a call - automatically tracked!
    const response = await anthropic.messages.create({
        model: 'claude-3-opus-20240229',
        max_tokens: 1024,
        messages: [
            { role: 'user', content: 'What is the capital of France?' },
        ],
    });

    console.log('Response:', response.content[0]?.text);
    console.log('Interaction recorded in Eyesite!');
}

main().catch(console.error);
