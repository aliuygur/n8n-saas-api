import { useState, useEffect } from "react";
import type { Route } from "./+types/dashboard";
import { isAuthenticated, logout } from "~/utils/auth";
import { API_BASE_URL } from "~/lib/config";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "Dashboard - instol.cloud" },
    { name: "description", content: "Manage your n8n instances" },
  ];
}

interface Instance {
  id: string;
  instance_url: string;
  status: string;
  created_at: string;
}

export default function Dashboard() {
  const [instances, setInstances] = useState<Instance[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleteModal, setDeleteModal] = useState<{ show: boolean; instance: Instance | null }>({
    show: false,
    instance: null,
  });
  const [deleteConfirmation, setDeleteConfirmation] = useState("");

  useEffect(() => {
    // Check authentication
    if (!isAuthenticated()) {
      window.location.href = "/login";
      return;
    }
    fetchInstances();
  }, []);

  const fetchInstances = async () => {
    try {
      const token = localStorage.getItem("jwt_token");
      const response = await fetch(`${API_BASE_URL}/api/instances`, {
        headers: {
          "Authorization": `Bearer ${token}`,
        },
      });
      if (response.ok) {
        const data = await response.json();
        setInstances(data.instances || []);
      } else if (response.status === 401) {
        // Unauthorized, redirect to login
        window.location.href = "/login";
      }
    } catch (error) {
      console.error("Failed to fetch instances:", error);
    } finally {
      setLoading(false);
    }
  };

  const deleteInstance = async (id: string) => {
    try {
      const token = localStorage.getItem("jwt_token");
      const response = await fetch(`${API_BASE_URL}/api/instances/${id}`, {
        method: "DELETE",
        headers: {
          "Authorization": `Bearer ${token}`,
        },
      });
      if (response.ok) {
        setDeleteModal({ show: false, instance: null });
        setDeleteConfirmation("");
        fetchInstances();
      }
    } catch (error) {
      console.error("Failed to delete instance:", error);
    }
  };

  const openDeleteModal = (instance: Instance) => {
    setDeleteModal({ show: true, instance });
    setDeleteConfirmation("");
  };

  const closeDeleteModal = () => {
    setDeleteModal({ show: false, instance: null });
    setDeleteConfirmation("");
  };

  const handleDelete = () => {
    if (deleteModal.instance) {
      deleteInstance(deleteModal.instance.id);
    }
  };

  const getSubdomain = (url: string) => {
    // Extract subdomain from https://subdomain.instol.cloud
    return url.replace('https://', '').split('.')[0];
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
            <button
              onClick={() => logout()}
              className="text-gray-400 hover:text-white transition-colors font-medium"
            >
              Logout
            </button>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="mb-8 flex justify-between items-start">
          <div>
            <h2 className="text-3xl font-bold text-white mb-2">Your Instances</h2>
            <p className="text-gray-400">Manage your workflow automation instances</p>
          </div>
          <a
            href="/create-instance"
            className="bg-indigo-600 text-white px-6 py-2 rounded-lg hover:bg-indigo-500 transition-all font-medium shadow-lg shadow-indigo-500/20"
          >
            Create Instance
          </a>
        </div>

        {loading ? (
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500"></div>
          </div>
        ) : instances.length === 0 ? (
          <div className="bg-gray-900/50 rounded-2xl p-16 text-center border border-gray-800 backdrop-blur-sm">
            <div className="max-w-md mx-auto">
              <div className="relative mb-8">
                <div className="absolute inset-0 flex items-center justify-center">
                  <div className="w-32 h-32 bg-indigo-500/10 rounded-full blur-2xl"></div>
                </div>
                <svg className="w-20 h-20 text-gray-700 mx-auto relative" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                </svg>
              </div>
              
              <h3 className="text-2xl font-bold text-white mb-3">
                No instances yet
              </h3>
              <p className="text-gray-400 mb-8 leading-relaxed">
                Get started by deploying your first workflow automation instance. It takes less than a minute to get up and running.
              </p>
              
              <a
                href="/create-instance"
                className="inline-flex items-center gap-2 bg-indigo-600 text-white px-8 py-3.5 rounded-lg hover:bg-indigo-500 transition-all font-semibold shadow-lg shadow-indigo-500/30 hover:shadow-indigo-500/40 hover:scale-105"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Create Your First Instance
              </a>
              
              <div className="mt-10 pt-8 border-t border-gray-800">
                <div className="grid grid-cols-3 gap-6 text-sm">
                  <div>
                    <div className="text-indigo-400 font-semibold mb-1">Fast Deploy</div>
                    <div className="text-gray-500">Ready in minutes</div>
                  </div>
                  <div>
                    <div className="text-indigo-400 font-semibold mb-1">Auto SSL</div>
                    <div className="text-gray-500">Secure by default</div>
                  </div>
                  <div>
                    <div className="text-indigo-400 font-semibold mb-1">Scalable</div>
                    <div className="text-gray-500">Cloud-powered</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {instances.map((instance) => (
              <div
                key={instance.id}
                className="bg-gray-900/50 rounded-xl p-6 border border-gray-800 hover:border-gray-700 backdrop-blur-sm transition-all"
              >
                <div className="flex justify-between items-start mb-4">
                  <div>
                    <h3 className="text-xl font-semibold text-white mb-2">
                      {instance.instance_url.replace('https://', '')}
                    </h3>
                    <span
                      className={`inline-block px-3 py-1 rounded-full text-xs font-medium ${
                        instance.status === "running"
                          ? "bg-green-500/10 text-green-400 border border-green-500/20"
                          : "bg-yellow-500/10 text-yellow-400 border border-yellow-500/20"
                      }`}
                    >
                      {instance.status}
                    </span>
                  </div>
                </div>
                <p className="text-gray-400 text-sm mb-6">
                  Created {new Date(instance.created_at).toLocaleDateString()}
                </p>
                <div className="flex gap-2">
                  <a
                    href={instance.instance_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex-1 bg-indigo-600 hover:bg-indigo-500 text-white text-center py-2 rounded-lg transition-colors text-sm font-medium shadow-lg shadow-indigo-500/20 flex items-center justify-center gap-2"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                    </svg>
                    Open Instance
                  </a>
                  <button
                    onClick={() => openDeleteModal(instance)}
                    className="px-4 bg-red-500/10 hover:bg-red-500/20 text-red-400 rounded-lg transition-colors text-sm font-medium border border-red-500/20"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Delete Confirmation Modal */}
        {deleteModal.show && deleteModal.instance && (
          <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
            <div className="bg-gray-900 rounded-2xl border border-gray-800 max-w-md w-full p-6 shadow-2xl">
              <div className="flex items-start gap-4 mb-4">
                <div className="flex-shrink-0 w-12 h-12 rounded-full bg-red-500/10 flex items-center justify-center border border-red-500/20">
                  <svg className="w-6 h-6 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                </div>
                <div className="flex-1">
                  <h3 className="text-xl font-semibold text-white mb-1">Delete Instance</h3>
                  <p className="text-gray-400 text-sm">This action cannot be undone.</p>
                </div>
              </div>

              <div className="mb-6">
                <p className="text-gray-300 mb-4">
                  You are about to delete <span className="font-semibold text-white">{deleteModal.instance.instance_url.replace('https://', '')}</span>
                </p>
                <p className="text-gray-400 text-sm mb-3">
                  Type <span className="font-mono font-semibold text-white">{getSubdomain(deleteModal.instance.instance_url)}</span> to confirm:
                </p>
                <input
                  type="text"
                  value={deleteConfirmation}
                  onChange={(e) => setDeleteConfirmation(e.target.value)}
                  className="w-full bg-gray-950 border border-gray-700 rounded-lg px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:border-red-500 focus:ring-2 focus:ring-red-500/20 transition-all"
                  placeholder="Type subdomain to confirm"
                  autoFocus
                />
              </div>

              <div className="flex gap-3">
                <button
                  onClick={closeDeleteModal}
                  className="flex-1 bg-gray-800 hover:bg-gray-700 text-white py-2.5 rounded-lg transition-colors font-medium"
                >
                  Cancel
                </button>
                <button
                  onClick={handleDelete}
                  disabled={deleteConfirmation !== getSubdomain(deleteModal.instance.instance_url)}
                  className="flex-1 bg-red-600 hover:bg-red-500 text-white py-2.5 rounded-lg transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-red-600"
                >
                  Delete Instance
                </button>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
