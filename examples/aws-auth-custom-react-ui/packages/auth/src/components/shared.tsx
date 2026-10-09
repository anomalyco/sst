/** @jsxImportSource react */
import { ReactNode } from "react";

export function CardWrapper({ children }: { children: ReactNode }) {
  return (
    <div className="w-full min-h-screen flex items-center justify-center p-4 relative overflow-hidden bg-slate-950">
      {/* Background decorations */}
      <div className="absolute top-1/4 left-1/4 -translate-x-1/2 -translate-y-1/2 w-125 h-125 bg-indigo-500/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-1/4 right-1/4 translate-x-1/2 translate-y-1/2 w-125 h-125 bg-purple-500/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute inset-0 bg-[radial-gradient(slate-900_1px,transparent_1px)] bg-size-[24px_24px] opacity-30 pointer-events-none" />

      {/* Main card */}
      <div className="w-full max-w-md bg-slate-900/60 border border-slate-800/80 rounded-3xl p-8 backdrop-blur-xl shadow-2xl relative overflow-hidden group">
        <div className="absolute top-0 left-0 right-0 h-px bg-linear-to-r from-transparent via-indigo-500/20 to-transparent" />
        
        {/* Brand header */}
        <div className="flex items-center gap-3 mb-8">
          <div className="w-8 h-8 rounded-xl bg-linear-to-tr from-indigo-500 to-purple-600 flex items-center justify-center shadow-md">
            <span className="font-bold text-white text-base">A</span>
          </div>
          <span className="font-semibold text-slate-200 tracking-tight text-sm">Identity Hub</span>
        </div>

        {children}
      </div>
    </div>
  );
}

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
}

export function FormInput({ label, id, ...props }: InputProps) {
  return (
    <div>
      <label className="block text-slate-300 text-xs font-semibold uppercase tracking-wider mb-2" htmlFor={id}>
        {label}
      </label>
      <input
        id={id}
        {...props}
        className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-3 text-white placeholder-slate-600 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all font-medium text-sm"
      />
    </div>
  );
}

export function FormButton({ children, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...props}
      className="w-full bg-gradient-to-r from-indigo-500 to-purple-600 hover:from-indigo-400 hover:to-purple-500 text-white font-medium py-3.5 px-4 rounded-xl transition-all shadow-lg shadow-indigo-500/25 active:scale-[0.98]"
    >
      {children}
    </button>
  );
}

export function FormAlert({ title, message }: { title: string; message: string }) {
  return (
    <div className="bg-red-500/10 border border-red-500/20 text-red-200 text-sm rounded-xl p-4 flex items-start gap-3">
      <svg className="w-5 h-5 text-red-400 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <div>
        <span className="font-semibold text-xs uppercase tracking-wider block text-red-400">{title}</span>
        <p className="text-xs text-red-300/80 mt-1">{message}</p>
      </div>
    </div>
  );
}
