import type { ReactNode } from "react";
import { NavLink } from "react-router";

const linkClass = ({ isActive }: { isActive: boolean }) =>
  `px-3 py-2 rounded-md text-sm font-medium ${
    isActive ? "bg-slate-900 text-white" : "text-slate-700 hover:bg-slate-100"
  }`;

export function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-slate-50">
      <nav className="border-b border-slate-200 bg-white px-4 py-3">
        <div className="mx-auto flex max-w-3xl gap-2">
          <NavLink to="/" className={linkClass} end>
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
      <main className="mx-auto max-w-3xl px-4 py-6">{children}</main>
    </div>
  );
}
