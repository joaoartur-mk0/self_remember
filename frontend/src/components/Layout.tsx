import type { ReactNode } from "react";
import { NavLink } from "react-router";

const linkClass = ({ isActive }: { isActive: boolean }) =>
  `px-3 py-2 rounded-md text-sm font-medium transition-colors ${
    isActive ? "bg-indigo-500 text-white" : "text-slate-300 hover:bg-slate-800 hover:text-slate-100"
  }`;

export function Layout({ children, wide = false }: { children: ReactNode; wide?: boolean }) {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <nav className="border-b border-slate-800 bg-slate-900 px-4 py-3">
        <div className={`mx-auto flex items-center gap-2 ${wide ? "max-w-5xl" : "max-w-3xl"}`}>
          <span className="mr-2 text-sm font-semibold tracking-tight text-slate-100">Self Remember</span>
          <NavLink to="/" className={linkClass} end>
            Painel
          </NavLink>
          <NavLink to="/hoje" className={linkClass}>
            Hoje
          </NavLink>
          <NavLink to="/compromissos" className={linkClass}>
            Compromissos
          </NavLink>
          <NavLink to="/ajustes" className={linkClass}>
            Ajustes
          </NavLink>
        </div>
      </nav>
      <main className={`mx-auto px-4 py-6 ${wide ? "max-w-5xl" : "max-w-3xl"}`}>{children}</main>
    </div>
  );
}
