import { createFileRoute, Link } from '@tanstack/react-router'

export const Route = createFileRoute('/demo/start/ssr/')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <div
      className="flex items-center justify-center min-h-screen bg-gradient-to-br from-zinc-900 to-black p-4 text-white"
      style={{
        backgroundImage:
          'radial-gradient(50% 50% at 20% 60%, #1a1a1a 0%, #0a0a0a 50%, #000000 100%)',
      }}
    >
      <div className="w-full max-w-2xl p-8 rounded-xl backdrop-blur-md bg-black/50 shadow-xl border-8 border-black/10">
        <h1 className="text-4xl font-bold mb-8 text-center bg-gradient-to-r from-pink-500 via-purple-500 to-green-400 bg-clip-text text-transparent">
          SSR Demos
        </h1>
        <div className="flex flex-col gap-4">
          <Link
            to="/demo/start/ssr/spa-mode"
            className="text-2xl font-bold py-6 px-8 rounded-lg bg-gradient-to-r from-pink-600 to-pink-500 hover:from-pink-700 hover:to-pink-600 text-white text-center shadow-lg transform transition-all hover:scale-105 hover:shadow-pink-500/50 border-2 border-pink-400"
          >
            SPA Mode
          </Link>
          <Link
            to="/demo/start/ssr/full-ssr"
            className="text-2xl font-bold py-6 px-8 rounded-lg bg-gradient-to-r from-purple-600 to-purple-500 hover:from-purple-700 hover:to-purple-600 text-white text-center shadow-lg transform transition-all hover:scale-105 hover:shadow-purple-500/50 border-2 border-purple-400"
          >
            Full SSR
          </Link>
          <Link
            to="/demo/start/ssr/data-only"
            className="text-2xl font-bold py-6 px-8 rounded-lg bg-gradient-to-r from-green-500 to-emerald-500 hover:from-green-600 hover:to-emerald-600 text-white text-center shadow-lg transform transition-all hover:scale-105 hover:shadow-green-500/50 border-2 border-green-400"
          >
            Data Only
          </Link>
        </div>
      </div>
    </div>
  )
}
