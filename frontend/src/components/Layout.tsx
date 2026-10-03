import { useEffect, useState, type ReactNode } from "react";
import { useLocation } from "react-router";
import { Sidebar } from "./Sidebar";

// Barra lateral fixa à esquerda em telas médias para cima; em telas
// pequenas vira uma faixa no topo com botão que abre/fecha o menu.
export function Layout({ children, wide = false }: { children: ReactNode; wide?: boolean }) {
  const { pathname } = useLocation();
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    setMenuOpen(false);
  }, [pathname]);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 md:flex">
      <header className="flex items-center justify-between border-b border-slate-800 bg-slate-900 px-4 py-3 md:hidden">
        <span className="text-sm font-semibold tracking-tight">Self Remember</span>
        <button
          type="button"
          onClick={() => setMenuOpen((o) => !o)}
          aria-expanded={menuOpen}
          aria-controls="sidebar"
          className="rounded-md border border-slate-700 px-3 py-1 text-sm text-slate-300 hover:bg-slate-800"
        >
          {menuOpen ? "Fechar" : "Menu"}
        </button>
      </header>

      <aside
        id="sidebar"
        className={`${menuOpen ? "block" : "hidden"} border-b border-slate-800 bg-slate-900 md:sticky md:top-0 md:block md:h-screen md:w-64 md:shrink-0 md:overflow-y-auto md:border-b-0 md:border-r`}
      >
        <Sidebar />
      </aside>

      <main className="min-w-0 flex-1 px-4 py-6 md:px-8">
        <div className={`mx-auto ${wide ? "max-w-5xl" : "max-w-3xl"}`}>{children}</div>
      </main>
    </div>
  );
}
