import { useEffect, useState } from "react";
import { NavLink, useLocation } from "react-router";

type Item = { to: string; label: string; end?: boolean };
type Group = { label: string; prefix: string; items: Item[] };

const TOP_ITEMS: Item[] = [
  { to: "/", label: "Painel", end: true },
  { to: "/hoje", label: "Hoje" },
  { to: "/compromissos", label: "Compromissos" },
];

// Grupos com submenu. Os itens aparecem logo abaixo do cabeçalho do grupo,
// dentro da própria barra (acordeão) — novos cadastros entram em
// "Cadastros" sem mexer em mais nada.
const GROUPS: Group[] = [
  {
    label: "Cadastros",
    prefix: "/cadastros",
    items: [{ to: "/cadastros/areas", label: "Áreas" }],
  },
  {
    label: "Configurações",
    prefix: "/configuracoes",
    items: [
      { to: "/configuracoes/preferencias", label: "Preferências" },
      { to: "/configuracoes/canais", label: "Canais" },
      { to: "/configuracoes/conexao", label: "Conexão WhatsApp" },
    ],
  },
];

const baseLink = "block rounded-md px-3 py-2 text-sm font-medium transition-colors";
const linkClass = ({ isActive }: { isActive: boolean }) =>
  `${baseLink} ${isActive ? "bg-indigo-500 text-white" : "text-slate-300 hover:bg-slate-800 hover:text-slate-100"}`;
const subLinkClass = ({ isActive }: { isActive: boolean }) =>
  `${baseLink} ${isActive ? "bg-slate-800 text-indigo-300" : "text-slate-400 hover:bg-slate-800 hover:text-slate-100"}`;

function NavGroup({ group }: { group: Group }) {
  const { pathname } = useLocation();
  const active = pathname.startsWith(group.prefix);
  const [open, setOpen] = useState(active);
  const panelID = `submenu-${group.prefix.slice(1)}`;

  // Navegar para uma rota do grupo (link direto, redirect) o abre.
  useEffect(() => {
    if (active) setOpen(true);
  }, [active]);

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        aria-controls={panelID}
        className={`flex w-full items-center justify-between rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-slate-800 ${
          active ? "text-indigo-300" : "text-slate-300 hover:text-slate-100"
        }`}
      >
        {group.label}
        <svg
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
          className={`h-4 w-4 transition-transform ${open ? "rotate-180" : ""}`}
        >
          <path
            fillRule="evenodd"
            d="M5.23 7.21a.75.75 0 011.06.02L10 11.17l3.71-3.94a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z"
            clipRule="evenodd"
          />
        </svg>
      </button>
      {open && (
        <div id={panelID} className="ml-3 mt-1 space-y-1 border-l border-slate-800 pl-2">
          {group.items.map((item) => (
            <NavLink key={item.to} to={item.to} className={subLinkClass}>
              {item.label}
            </NavLink>
          ))}
        </div>
      )}
    </div>
  );
}

export function Sidebar() {
  return (
    <nav aria-label="Principal" className="flex flex-col gap-1 p-3">
      <span className="mb-3 px-3 pt-1 text-sm font-semibold tracking-tight text-slate-100">Self Remember</span>
      {TOP_ITEMS.map((item) => (
        <NavLink key={item.to} to={item.to} end={item.end} className={linkClass}>
          {item.label}
        </NavLink>
      ))}
      {GROUPS.map((group) => (
        <NavGroup key={group.prefix} group={group} />
      ))}
    </nav>
  );
}
