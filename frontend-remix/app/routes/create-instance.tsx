import { useState, useEffect } from "react";
import type { Route } from "./+types/create-instance";
import { isAuthenticated } from "~/utils/auth";
import { API_BASE_URL } from "~/lib/config";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Create Instance - instol.cloud" },
    { name: "description", content: "Deploy a new n8n instance on Google Cloud" },
  ];
}

export default function CreateInstance() {
  const [subdomain, setSubdomain] = useState("");
  const [region, setRegion] = useState("us-central");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [checkingAvailability, setCheckingAvailability] = useState(false);
  const [availabilityMessage, setAvailabilityMessage] = useState("");
  const [isAvailable, setIsAvailable] = useState<boolean | null>(null);

  useEffect(() => {
    // Check authentication
    if (!isAuthenticated()) {
      window.location.href = "/login";
    }
  }, []);

  // Debounce subdomain availability check
  useEffect(() => {
    if (!subdomain || subdomain.length < 3) {
      setAvailabilityMessage("");
      setIsAvailable(null);
      return;
    }

    const timeoutId = setTimeout(async () => {
      setCheckingAvailability(true);
      try {
        const token = localStorage.getItem("jwt_token");
        const response = await fetch(`${API_BASE_URL}/api/instances/check-subdomain`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Authorization": `Bearer ${token}`,
          },
          body: JSON.stringify({ subdomain }),
        });

        if (response.ok) {
          const data = await response.json();
          setIsAvailable(data.available);
          setAvailabilityMessage(data.message);
        }
      } catch (err) {
        console.error("Failed to check subdomain availability:", err);
      } finally {
        setCheckingAvailability(false);
      }
    }, 500); // 500ms debounce

    return () => clearTimeout(timeoutId);
  }, [subdomain]);
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const token = localStorage.getItem("jwt_token");
      const response = await fetch(`${API_BASE_URL}/api/instances`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
        },
        body: JSON.stringify({ subdomain, region }),
      });

      if (response.ok) {
        window.location.href = "/dashboard";
      } else {
        const data = await response.json();
        setError(data.error || "Failed to create instance");
      }
    } catch (err) {
      setError("Failed to create instance. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-950">
      <nav className="border-b border-gray-800 bg-gray-900/50 backdrop-blur-lg">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <a href="/" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
              <svg className="w-8 h-8 text-indigo-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              <h1 className="text-2xl font-bold text-white">instol.cloud</h1>
            </a>
            <a
              href="/dashboard"
              className="text-gray-300 hover:text-white transition-colors font-medium"
            >
              Back to Dashboard
            </a>
          </div>
        </div>
      </nav>

      <main className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="bg-gray-900/50 rounded-2xl p-8 border border-gray-800 backdrop-blur-sm">
          <h2 className="text-3xl font-bold text-white mb-2">
            Deploy New Instance
          </h2>
          <p className="text-gray-400 mb-8">
            Create your own n8n workflow automation instance on Google Cloud
          </p>

          <form onSubmit={handleSubmit}>
            <div className="mb-6">
              <label
                htmlFor="subdomain"
                className="block text-sm font-medium text-gray-300 mb-2"
              >
                Instance Domain
              </label>
              <div className="relative">
                <input
                  type="text"
                  id="subdomain"
                  value={subdomain}
                  onChange={(e) => setSubdomain(e.target.value.toLowerCase())}
                  className={`w-full bg-gray-950 border rounded-lg pl-4 pr-32 py-3 text-white placeholder-gray-500 focus:outline-none transition-all ${
                    isAvailable === true
                      ? "border-green-500 focus:border-green-500 focus:ring-2 focus:ring-green-500/20"
                      : isAvailable === false
                      ? "border-red-500 focus:border-red-500 focus:ring-2 focus:ring-red-500/20"
                      : "border-gray-700 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20"
                  }`}
                  placeholder="myapp"
                  required
                />
                <div className="absolute inset-y-0 right-0 flex items-center pr-4 gap-2">
                  {checkingAvailability && (
                    <svg className="animate-spin h-4 w-4 text-indigo-500" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                  )}
                  {!checkingAvailability && isAvailable === true && (
                    <svg className="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                  )}
                  {!checkingAvailability && isAvailable === false && (
                    <svg className="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  )}
                  <span className="text-gray-500 font-medium pointer-events-none">.instol.cloud</span>
                </div>
              </div>
              {availabilityMessage && (
                <p className={`text-sm mt-2 ${isAvailable ? "text-green-400" : "text-red-400"}`}>
                  {availabilityMessage}
                </p>
              )}
              {!availabilityMessage && (
                <p className="text-gray-500 text-sm mt-2">
                  Your instance will be available at <span className="text-gray-400 font-mono">{subdomain || 'myapp'}.instol.cloud</span>
                </p>
              )}
            </div>

            <div className="mb-6">
              <label className="block text-sm font-medium text-gray-300 mb-3">
                Region
              </label>
              <div className="space-y-3">
                <label className="flex items-center p-4 bg-gray-950 border border-gray-700 rounded-lg cursor-pointer hover:border-indigo-500 transition-all">
                  <input
                    type="radio"
                    name="region"
                    value="us-central"
                    checked={region === "us-central"}
                    onChange={(e) => setRegion(e.target.value)}
                    className="w-4 h-4 text-indigo-600 bg-gray-900 border-gray-700 focus:ring-indigo-500 focus:ring-2"
                  />
                  <div className="ml-3 flex-1">
                    <div className="flex items-center justify-between">
                      <span className="text-white font-medium">US Central</span>
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-500/10 text-green-400 border border-green-500/20">
                        Active
                      </span>
                    </div>
                    <p className="text-gray-400 text-sm mt-1">Hosted in Iowa, USA</p>
                  </div>
                </label>

                <label className="flex items-center p-4 bg-gray-950 border border-gray-700 rounded-lg opacity-60 cursor-not-allowed">
                  <input
                    type="radio"
                    name="region"
                    value="europe"
                    disabled
                    className="w-4 h-4 text-indigo-600 bg-gray-900 border-gray-700"
                  />
                  <div className="ml-3 flex-1">
                    <div className="flex items-center justify-between">
                      <span className="text-white font-medium">Europe</span>
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-500/10 text-yellow-400 border border-yellow-500/20">
                        Soon
                      </span>
                    </div>
                    <p className="text-gray-400 text-sm mt-1">Coming soon</p>
                  </div>
                </label>

                <label className="flex items-center p-4 bg-gray-950 border border-gray-700 rounded-lg opacity-60 cursor-not-allowed">
                  <input
                    type="radio"
                    name="region"
                    value="asia"
                    disabled
                    className="w-4 h-4 text-indigo-600 bg-gray-900 border-gray-700"
                  />
                  <div className="ml-3 flex-1">
                    <div className="flex items-center justify-between">
                      <span className="text-white font-medium">Asia</span>
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-500/10 text-yellow-400 border border-yellow-500/20">
                        Soon
                      </span>
                    </div>
                    <p className="text-gray-400 text-sm mt-1">Coming soon</p>
                  </div>
                </label>
              </div>
            </div>

            {error && (
              <div className="mb-6 bg-red-500/10 border border-red-500/20 rounded-lg p-4">
                <p className="text-red-400 text-sm">{error}</p>
              </div>
            )}

            <button
              type="submit"
              disabled={loading || isAvailable === false || !subdomain}
              className="w-full bg-indigo-600 text-white font-semibold py-3 px-4 rounded-lg hover:bg-indigo-500 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-indigo-500/20"
            >
              {loading ? "Deploying..." : "Deploy Instance"}
            </button>
          </form>

          <div className="mt-8 pt-8 border-t border-gray-800">
            <h3 className="text-lg font-semibold text-white mb-4">
              What happens next?
            </h3>
            <ul className="space-y-3 text-gray-300">
              <li className="flex items-start gap-3">
                <svg
                  className="w-6 h-6 text-green-500 flex-shrink-0 mt-0.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
                <span>Your instance will be deployed on Google Cloud Platform</span>
              </li>
              <li className="flex items-start gap-3">
                <svg
                  className="w-6 h-6 text-green-500 flex-shrink-0 mt-0.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
                <span>SSL certificate will be automatically configured and renewed</span>
              </li>
              <li className="flex items-start gap-3">
                <svg
                  className="w-6 h-6 text-green-500 flex-shrink-0 mt-0.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
                <span>You'll be able to access your n8n instance within a few minutes</span>
              </li>
            </ul>
          </div>
        </div>
      </main>
    </div>
  );
}
