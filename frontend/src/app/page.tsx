'use client';

import Link from 'next/link';

export default function HomePage() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50">
      {/* Hero Section */}
      <div className="relative overflow-hidden">
        <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGRlZnM+PHBhdHRlcm4gaWQ9ImdyaWQiIHdpZHRoPSI2MCIgaGVpZ2h0PSI2MCIgcGF0dGVyblVuaXRzPSJ1c2VyU3BhY2VPblVzZSI+PHBhdGggZD0iTSAxMCAwIEwgMCAwIDAgMTAiIGZpbGw9Im5vbmUiIHN0cm9rZT0icmdiYSgwLDAsMCwwLjAyKSIgc3Ryb2tlLXdpZHRoPSIxIi8+PC9wYXR0ZXJuPjwvZGVmcz48cmVjdCB3aWR0aD0iMTAwJSIgaGVpZ2h0PSIxMDAlIiBmaWxsPSJ1cmwoI2dyaWQpIi8+PC9zdmc+')] opacity-40"></div>

        <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-20 pb-24">
          {/* Logo */}
          <div className="text-center mb-8">
            <div className="inline-flex items-center space-x-3">
              <div className="w-12 h-12 bg-gradient-to-br from-blue-600 to-purple-600 rounded-xl flex items-center justify-center text-white font-bold text-2xl shadow-lg">
                E
              </div>
              <h1 className="text-4xl font-bold text-slate-900">Eyesite</h1>
            </div>
          </div>

          {/* Hero Text */}
          <div className="text-center max-w-3xl mx-auto mb-12">
            <h2 className="text-5xl sm:text-6xl font-extrabold text-slate-900 mb-6">
              See Everything
              <br />
              <span className="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                Your AI Agents Do
              </span>
            </h2>
            <p className="text-xl text-slate-600 mb-8">
              Open-source observability platform for AI agents. Track every interaction, optimize costs, and build better AI systems.
            </p>
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
              <Link
                href="/register"
                className="px-8 py-4 bg-gradient-to-r from-blue-600 to-purple-600 text-white font-semibold rounded-xl shadow-lg hover:shadow-xl transform hover:scale-105 transition-all duration-200"
              >
                Get Started Free
              </Link>
              <Link
                href="/login"
                className="px-8 py-4 bg-white text-slate-900 font-semibold rounded-xl shadow-md hover:shadow-lg border-2 border-slate-200 hover:border-blue-200 transition-all duration-200"
              >
                Sign In
              </Link>
            </div>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-8 max-w-4xl mx-auto">
            <div className="text-center">
              <div className="text-3xl font-bold text-slate-900">3</div>
              <div className="text-sm text-slate-600 mt-1">AI Providers</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-slate-900">100%</div>
              <div className="text-sm text-slate-600 mt-1">Open Source</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-slate-900">Free</div>
              <div className="text-sm text-slate-600 mt-1">Self-Hosted</div>
            </div>
            <div className="text-center">
              <div className="text-3xl font-bold text-slate-900">2s</div>
              <div className="text-sm text-slate-600 mt-1">Setup Time</div>
            </div>
          </div>
        </div>
      </div>

      {/* Features */}
      <div className="py-24 bg-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h3 className="text-3xl font-bold text-slate-900 mb-4">Everything You Need</h3>
            <p className="text-lg text-slate-600">Track, analyze, and optimize your AI agents in real-time</p>
          </div>

          <div className="grid md:grid-cols-3 gap-8">
            <div className="p-6 rounded-xl border-2 border-slate-100 hover:border-blue-200 hover:shadow-lg transition">
              <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center text-2xl mb-4">
                📊
              </div>
              <h4 className="text-xl font-semibold text-slate-900 mb-2">Real-Time Analytics</h4>
              <p className="text-slate-600">
                Monitor costs, tokens, and performance across all your AI agents in one dashboard.
              </p>
            </div>

            <div className="p-6 rounded-xl border-2 border-slate-100 hover:border-purple-200 hover:shadow-lg transition">
              <div className="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center text-2xl mb-4">
                🤖
              </div>
              <h4 className="text-xl font-semibold text-slate-900 mb-2">Multi-Provider</h4>
              <p className="text-slate-600">
                Works with OpenAI, Anthropic, Google Gemini, and more. One platform for everything.
              </p>
            </div>

            <div className="p-6 rounded-xl border-2 border-slate-100 hover:border-green-200 hover:shadow-lg transition">
              <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center text-2xl mb-4">
                ⚡
              </div>
              <h4 className="text-xl font-semibold text-slate-900 mb-2">2-Line Setup</h4>
              <p className="text-slate-600">
                Wrap your AI client and start tracking. That's it. No complex configuration needed.
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Code Example */}
      <div className="py-24 bg-slate-50">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-12">
            <h3 className="text-3xl font-bold text-slate-900 mb-4">Start in Seconds</h3>
            <p className="text-lg text-slate-600">Literally two lines of code</p>
          </div>

          <div className="bg-slate-900 rounded-xl p-8 shadow-2xl">
            <div className="flex items-center space-x-2 mb-4">
              <div className="w-3 h-3 rounded-full bg-red-500"></div>
              <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
              <div className="w-3 h-3 rounded-full bg-green-500"></div>
            </div>
            <pre className="text-green-400 font-mono text-sm">
              <code>{`import { Eyesite } from '@eyesite/sdk';
import OpenAI from 'openai';

const eyesite = new Eyesite({ apiKey: 'your-key' });
const openai = new OpenAI();

// Just wrap it - that's all!
eyesite.wrapOpenAI(openai);

// Now all your calls are automatically tracked
const response = await openai.chat.completions.create({
  model: 'gpt-4',
  messages: [{ role: 'user', content: 'Hello!' }]
});`}</code>
            </pre>
          </div>
        </div>
      </div>

      {/* CTA */}
      <div className="py-24 bg-gradient-to-r from-blue-600 to-purple-600">
        <div className="max-w-4xl mx-auto text-center px-4 sm:px-6 lg:px-8">
          <h3 className="text-4xl font-bold text-white mb-6">
            Ready to see everything?
          </h3>
          <p className="text-xl text-blue-100 mb-8">
            Join developers building better AI systems with Eyesite
          </p>
          <Link
            href="/register"
            className="inline-block px-8 py-4 bg-white text-blue-600 font-semibold rounded-xl shadow-lg hover:shadow-xl transform hover:scale-105 transition-all duration-200"
          >
            Get Started Free
          </Link>
          <p className="text-blue-100 text-sm mt-4">
            Open source • Self-hosted • No credit card required
          </p>
        </div>
      </div>

      {/* Footer */}
      <div className="bg-slate-900 text-white py-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row justify-between items-center">
            <div className="flex items-center space-x-3 mb-4 md:mb-0">
              <div className="w-8 h-8 bg-gradient-to-br from-blue-600 to-purple-600 rounded-lg flex items-center justify-center font-bold">
                E
              </div>
              <span className="text-xl font-bold">Eyesite</span>
            </div>
            <div className="text-slate-400 text-sm">
              Apache 2.0 Licensed • Vibe-coded with ❤️
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
