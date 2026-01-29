/**
 * Example: Using Eyesite SDK with OpenAI
 */

import { Eyesite } from '@eyesite/sdk';
import OpenAI from 'openai';

async function main() {
    // Initialize Eyesite
    const eyesite = new Eyesite({
        apiKey: process.env.EYESITE_API_KEY || 'your-api-key',
        endpoint: 'http://localhost:8001',
        workspaceUuid: 'your-workspace-uuid',
    });

    // Create a session
    const sessionId = await eyesite.createSession('OpenAI Example');
    console.log('Created session:', sessionId);

    // Initialize OpenAI
    const openai = new OpenAI({
        apiKey: process.env.OPENAI_API_KEY,
    });

    // Wrap OpenAI for automatic tracking
    eyesite.wrapOpenAI(openai);

    // Make a call - automatically tracked!
    const response = await openai.chat.completions.create({
        model: 'gpt-4',
        messages: [
            { role: 'system', content: 'You are a helpful assistant.' },
            { role: 'user', content: 'What is the capital of France?' },
        ],
    });

    console.log('Response:', response.choices[0]?.message?.content);
    console.log('Interaction recorded in Eyesite!');
}

main().catch(console.error);
