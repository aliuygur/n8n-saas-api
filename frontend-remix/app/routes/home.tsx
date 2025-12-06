import type { Route } from "./+types/home";
import { redirect } from "react-router";

export function loader({ request }: Route.LoaderArgs) {
  // Check if user is authenticated via JWT cookie
  const cookieHeader = request.headers.get("Cookie");
  if (cookieHeader && cookieHeader.includes("jwt=")) {
    return redirect("/dashboard");
  }
  return null;
}

export function meta({}: Route.MetaArgs) {
  return [
    { title: "instol.cloud - One-Click Workflow Automation on GCP" },
    { name: "description", content: "Deploy your own workflow automation instance on Google Cloud in one click. Simple setup with unlimited workflows." },
  ];
}

export default function Home() {
  return (
    <div className="min-h-screen bg-gray-950">
      {/* Navigation */}
      <nav className="border-b border-gray-800 bg-gray-900/50 backdrop-blur-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <a href="/" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
              <svg className="w-8 h-8 text-indigo-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              <h1 className="text-2xl font-bold text-white">instol.cloud</h1>
            </a>
            <div className="flex gap-4">
              <a
                href="/login"
                className="text-gray-300 hover:text-white transition-colors px-4 py-2 font-medium"
              >
                Sign In
              </a>
              <a
                href="/login"
                className="bg-indigo-600 text-white px-6 py-2 rounded-lg hover:bg-indigo-500 transition-all font-medium shadow-lg shadow-indigo-500/20"
              >
                Get Started
              </a>
            </div>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="relative">
        {/* Background Pattern */}
        <div className="absolute inset-0 overflow-hidden">
          {/* Gradient orbs */}
          <div className="absolute top-20 -left-20 w-96 h-96 bg-indigo-500 rounded-full blur-3xl opacity-20"></div>
          <div className="absolute -bottom-20 right-10 w-[500px] h-[500px] bg-purple-500 rounded-full blur-3xl opacity-20"></div>
          <div className="absolute top-1/3 right-1/4 w-72 h-72 bg-pink-500 rounded-full blur-3xl opacity-15"></div>
          <div className="absolute bottom-1/3 left-1/3 w-80 h-80 bg-blue-500 rounded-full blur-3xl opacity-15"></div>
          
          {/* Dotted pattern overlay */}
          <div className="absolute inset-0 opacity-40" style={{
            backgroundImage: 'radial-gradient(circle, rgba(99, 102, 241, 0.4) 1px, transparent 1px)',
            backgroundSize: '30px 30px',
          }}></div>
          
          {/* Gradient fade */}
          <div className="absolute inset-0 bg-gradient-to-b from-transparent via-gray-950/50 to-gray-950"></div>
        </div>
        
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
          <div className="text-center py-16 md:py-24">
            <div className="inline-block bg-indigo-500/10 text-indigo-400 px-4 py-2 rounded-full text-sm font-medium mb-6 border border-indigo-500/20">
              Self-Hosted n8n Platform
            </div>
            <h2 className="text-4xl md:text-6xl font-bold text-white mb-6 leading-tight">
              Deploy Your Own n8n
              <br />
              <span className="text-indigo-400">
                in Minutes
              </span>
            </h2>
            <p className="text-lg md:text-xl text-gray-400 mb-12 max-w-3xl mx-auto leading-relaxed">
              Self-hosted n8n workflow automation on Google Cloud with automatic SSL and cloud infrastructure.
            </p>
            
            <a
              href="/login"
              className="inline-block bg-gradient-to-r from-indigo-600 to-indigo-500 text-white text-lg px-10 py-4 rounded-xl hover:from-indigo-500 hover:to-indigo-400 transition-all font-semibold shadow-lg shadow-indigo-500/30 hover:shadow-indigo-500/50 hover:scale-105"
            >
              Get Started →
            </a>
            
            <p className="text-sm text-gray-400 mt-6">
              <span className="text-green-400">3 days free trial</span>, then $9.99/month • No credit card required
            </p>
          </div>
        </div>
      </section>

      {/* Features */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8 py-16 md:py-20">
          <div className="bg-gray-900/50 rounded-xl p-8 border border-gray-800 backdrop-blur-sm hover:border-gray-700 transition-all">
            <div className="w-12 h-12 bg-indigo-500/10 rounded-lg flex items-center justify-center mb-4">
              <svg
                className="w-6 h-6 text-indigo-400"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 10V3L4 14h7v7l9-11h-7z"
                />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-3">
              Easy Deployment
            </h3>
            <p className="text-gray-400 leading-relaxed">
              Quickly set up your n8n instance without the hassle of manual configuration. One click and you're ready.
            </p>
          </div>

          <div className="bg-gray-900/50 rounded-xl p-8 border border-gray-800 backdrop-blur-sm hover:border-gray-700 transition-all">
            <div className="w-12 h-12 bg-green-500/10 rounded-lg flex items-center justify-center mb-4">
              <svg
                className="w-6 h-6 text-green-400"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
                />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-3">
              Automatic SSL
            </h3>
            <p className="text-gray-400 leading-relaxed">
              Secure your workflows with automatic SSL certificate generation and renewal. Always encrypted, always safe.
            </p>
          </div>

          <div className="bg-gray-900/50 rounded-xl p-8 border border-gray-800 backdrop-blur-sm hover:border-gray-700 transition-all">
            <div className="w-12 h-12 bg-blue-500/10 rounded-lg flex items-center justify-center mb-4">
              <svg
                className="w-6 h-6 text-blue-400"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"
                />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-3">
              Google Cloud Infrastructure
            </h3>
            <p className="text-gray-400 leading-relaxed">
              Leverage the reliability and scalability of Google Cloud Platform for your automation needs.
            </p>
          </div>

          <div className="bg-gray-900/50 rounded-xl p-8 border border-gray-800 backdrop-blur-sm hover:border-gray-700 transition-all">
            <div className="w-12 h-12 bg-purple-500/10 rounded-lg flex items-center justify-center mb-4">
              <svg
                className="w-6 h-6 text-purple-400"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"
                />
              </svg>
            </div>
            <h3 className="text-xl font-semibold text-white mb-3">
              Unlimited Workflows
            </h3>
            <p className="text-gray-400 leading-relaxed">
              Create as many workflows as you need without restrictions. Self-hosted means unlimited potential.
            </p>
          </div>
        </div>

        {/* Why Choose Us Section */}
        <div className="bg-gray-900/30 rounded-2xl p-12 md:p-16 my-16 border border-gray-800">
          <div className="text-center mb-12">
            <h3 className="text-3xl md:text-4xl font-bold text-white mb-4">
              Why Choose instol.cloud?
            </h3>
            <p className="text-lg text-gray-400 max-w-2xl mx-auto">
              Powerful workflow automation made accessible and affordable.
            </p>
          </div>
          
          <div className="grid md:grid-cols-3 gap-8 max-w-5xl mx-auto">
            <div className="text-center">
              <div className="text-4xl font-bold text-indigo-500 mb-2">⚡</div>
              <h4 className="font-semibold text-white mb-2">Fast Deployment</h4>
              <p className="text-gray-400 text-sm">Get up and running in minutes, not hours</p>
            </div>
            
            <div className="text-center">
              <div className="text-4xl font-bold text-green-500 mb-2">💰</div>
              <h4 className="font-semibold text-white mb-2">Affordable</h4>
              <p className="text-gray-400 text-sm">Transparent pricing without hidden costs</p>
            </div>
            
            <div className="text-center">
              <div className="text-4xl font-bold text-purple-500 mb-2">🔧</div>
              <h4 className="font-semibold text-white mb-2">Full Control</h4>
              <p className="text-gray-400 text-sm">Your instance, your data, your rules</p>
            </div>
          </div>
        </div>

        {/* CTA */}
        <div className="bg-gradient-to-r from-indigo-600 to-purple-600 rounded-2xl p-12 md:p-16 text-center my-20 shadow-2xl shadow-indigo-500/20">
          <h3 className="text-3xl md:text-4xl font-bold text-white mb-4">
            Ready to automate your workflows?
          </h3>
          <p className="text-indigo-100 mb-8 text-lg max-w-2xl mx-auto">
            Join developers and businesses automating their workflows on Google Cloud with instol.cloud
          </p>
          <a
            href="/login"
            className="inline-block bg-white text-indigo-600 px-8 py-4 rounded-lg hover:bg-gray-100 transition-all font-semibold text-lg shadow-lg"
          >
            Get Started Now
          </a>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-gray-800 bg-gray-900/50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <div className="flex items-center gap-2">
              <svg className="w-6 h-6 text-indigo-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              <span className="font-semibold text-white">instol.cloud</span>
            </div>
            <p className="text-gray-400 text-sm">
              © 2025 instol.cloud. Powered by Google Cloud Platform.
            </p>
          </div>
        </div>
      </footer>
    </div>
  );
}
