import { useState } from "react";
import { useAuth } from "./AuthContext";

function App() {
  const auth = useAuth();
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(false);

  async function callApi() {
    setLoading(true);
    try {
      const res = await fetch(`${import.meta.env.VITE_API_URL}me`, {
        headers: {
          Authorization: `Bearer ${await auth.getToken()}`,
        },
      });
      if (res.ok) {
        const data = await res.json();
        setStatus(JSON.stringify(data));
      } else {
        setStatus("Error: " + res.statusText);
      }
    } catch (e: unknown) {
      setStatus(`"Error: ${String(e)}`);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col justify-between selection:bg-indigo-500 selection:text-white">
      {/* Background patterns */}
      <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,var(--tw-gradient-stops))] from-indigo-900/20 via-slate-950 to-slate-950 pointer-events-none" />
      <div className="absolute top-0 left-0 right-0 h-1 bg-linear-to-r from-transparent via-indigo-500/20 to-transparent pointer-events-none" />

      {/* Header */}
      <header className="relative z-10 border-b border-slate-800/40 bg-slate-950/50 backdrop-blur-md px-6 py-4">
        <div className="max-w-6xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-linear-to-tr from-indigo-500 to-purple-600 flex items-center justify-center shadow-lg shadow-indigo-500/20">
              <span className="font-bold text-white text-lg">A</span>
            </div>
            <span className="font-semibold text-lg tracking-tight bg-linear-to-r from-indigo-200 to-slate-100 bg-clip-text text-transparent">
              Auth Demo
            </span>
          </div>
          {auth.loaded && auth.loggedIn && (
            <button
              onClick={auth.logout}
              className="text-sm font-medium text-slate-400 hover:text-slate-100 transition-colors"
            >
              Sign out
            </button>
          )}
        </div>
      </header>

      {/* Main Content */}
      <main className="relative z-10 grow flex items-center justify-center px-4 py-16">
        {!auth.loaded ? (
          <div className="flex flex-col items-center gap-3">
            <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
            <p className="text-slate-400 text-sm font-medium animate-pulse">
              Initializing session...
            </p>
          </div>
        ) : (
          <div className="w-full max-w-md bg-slate-900/60 border border-slate-800/80 rounded-3xl p-8 backdrop-blur-xl shadow-2xl relative overflow-hidden group">
            {/* Ambient glow */}
            <div className="absolute -top-24 -left-24 w-48 h-48 bg-indigo-500/10 rounded-full blur-3xl group-hover:bg-indigo-500/15 transition-all duration-500" />

            {auth.loggedIn ? (
              <div className="relative space-y-6">
                <div className="text-center">
                  <div className="inline-flex p-3 rounded-2xl bg-indigo-500/10 border border-indigo-500/20 text-indigo-400 mb-4">
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      fill="none"
                      viewBox="0 0 24 24"
                      strokeWidth={1.5}
                      stroke="currentColor"
                      className="w-6 h-6"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        d="M9 12.75 11.25 15 15 9.75M21 12c0 1.268-.63 2.39-1.593 3.068a3.745 3.745 0 0 1-1.043 3.296 3.745 3.745 0 0 1-3.296 1.043A3.745 3.745 0 0 1 12 21c-1.268 0-2.39-.63-3.068-1.593a3.746 3.746 0 0 1-3.296-1.043 3.745 3.745 0 0 1-1.043-3.296A3.745 3.745 0 0 1 3 12c0-1.268.63-2.39 1.593-3.068a3.745 3.745 0 0 1 1.043-3.296 3.746 3.746 0 0 1 3.296-1.043A3.746 3.746 0 0 1 12 3c1.268 0 2.39.63 3.068 1.593a3.746 3.746 0 0 1 3.296 1.043 3.746 3.746 0 0 1 1.043 3.296A3.745 3.745 0 0 1 21 12Z"
                      />
                    </svg>
                  </div>
                  <h2 className="text-2xl font-bold tracking-tight text-white">
                    Authenticated Successfully
                  </h2>
                  <p className="text-slate-400 text-sm mt-1">
                    Logged in as{" "}
                    <code className="text-indigo-300 font-mono text-xs px-2 py-0.5 bg-slate-950 rounded border border-slate-800">
                      {auth.userId}
                    </code>
                  </p>
                </div>

                <div className="h-1 bg-slate-800/60" />

                {status !== "" && (
                  <div className="bg-slate-950/60 border border-slate-800 rounded-xl p-4 overflow-x-auto">
                    <span className="text-slate-500 text-xs font-mono block mb-1">
                      API response:
                    </span>
                    <pre className="text-indigo-200 text-xs font-mono whitespace-pre-wrap">
                      {status}
                    </pre>
                  </div>
                )}

                <div className="flex flex-col gap-3">
                  <button
                    disabled={loading}
                    onClick={callApi}
                    className="w-full bg-indigo-600 hover:bg-indigo-500 text-white font-medium py-3 px-4 rounded-xl transition-all shadow-lg shadow-indigo-600/20 active:scale-[0.98] disabled:opacity-50 disabled:pointer-events-none flex items-center justify-center gap-2"
                  >
                    {loading && (
                      <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                    )}
                    Call Secure API
                  </button>
                  <button
                    onClick={auth.logout}
                    className="w-full bg-slate-800 hover:bg-slate-700 text-slate-100 font-medium py-3 px-4 rounded-xl transition-all active:scale-[0.98]"
                  >
                    Log out
                  </button>
                </div>
              </div>
            ) : (
              <div className="relative space-y-6 text-center">
                <div>
                  <h2 className="text-3xl font-extrabold tracking-tight text-white bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent">
                    Secure Web Portal
                  </h2>
                  <p className="text-slate-400 text-sm mt-2">
                    An example React application demonstrating AWS OpenAuth
                    integration with a custom React UI.
                  </p>
                </div>

                <div className="h-1 bg-slate-800/60" />

                <button
                  onClick={auth.login}
                  className="w-full bg-gradient-to-r from-indigo-500 to-purple-600 hover:from-indigo-400 hover:to-purple-500 text-white font-medium py-3.5 px-4 rounded-xl transition-all shadow-lg shadow-indigo-500/25 active:scale-[0.98]"
                >
                  Login with OAuth
                </button>
              </div>
            )}
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="relative z-10 border-t border-slate-800/40 py-6 px-6 text-center text-slate-500 text-xs">
        <p>© 2026 Antigravity. Powered by SST and OpenAuth.</p>
      </footer>
    </div>
  );
}

export default App;
