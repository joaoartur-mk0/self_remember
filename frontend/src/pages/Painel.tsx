import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";
import { Layout } from "../components/Layout";
import { api, ApiError } from "../lib/api";
import type { Dashboard, DashboardItem, NagProfile } from "../lib/types";

// Severidade = Perfil de Insistência (ADR-0004); o Degrau atual desempata
// (quanto mais cobrada sem resposta, mais "quente").
const SEVERITY: Record<NagProfile, { rank: number; label: string; badge: string }> = {
  implacavel: { rank: 3, label: "Implacável", badge: "border-red-500/30 bg-red-500/15 text-red-300" },
  normal: { rank: 2, label: "Normal", badge: "border-amber-500/30 bg-amber-500/15 text-amber-300" },
  leve: { rank: 1, label: "Leve", badge: "border-slate-600 bg-slate-700/40 text-slate-300" },
};

function bySeverity(a: DashboardItem, b: DashboardItem): number {
  return (
    SEVERITY[b.nag_profile].rank - SEVERITY[a.nag_profile].rank ||
    b.current_step - a.current_step ||
    a.due_at.localeCompare(b.due_at)
  );
}

const MAX_ITEMS_PER_DAY = 4;

export function Painel() {
  const [data, setData] = useState<Dashboard | null>(null);
  const [error, setError] = useState<"settings" | "generic" | null>(null);

  useEffect(() => {
    api
      .get<Dashboard>("/dashboard")
      .then(setData)
      .catch((err) => setError(err instanceof ApiError && err.status === 404 ? "settings" : "generic"));
  }, []);

  // Horário e datas sempre no fuso de Ajustes (o que o servidor usou para
  // agrupar os dias) — nunca no do navegador, que pode ser outro.
  const fmt = useMemo(() => {
    const tz = data?.timezone ?? "UTC";
    return {
      time: new Intl.DateTimeFormat("pt-BR", { timeZone: tz, hour: "2-digit", minute: "2-digit" }),
      dayMonth: new Intl.DateTimeFormat("pt-BR", { timeZone: tz, day: "2-digit", month: "2-digit" }),
    };
  }, [data?.timezone]);

  if (error === "settings") {
    return (
      <Layout wide>
        <p className="text-slate-400">
          Defina seu fuso horário em <Link to="/ajustes" className="text-indigo-400 hover:underline">Ajustes</Link> para
          ver o painel.
        </p>
      </Layout>
    );
  }
  if (error) {
    return (
      <Layout wide>
        <p className="text-red-400">Não foi possível carregar o painel.</p>
      </Layout>
    );
  }
  if (!data) {
    return (
      <Layout wide>
        <p className="text-slate-400">Carregando...</p>
      </Layout>
    );
  }

  const today = data.days[0];
  const focus = [...data.overdue, ...today.items].sort(bySeverity);
  const weekItems = data.days.flatMap((d) => d.items);
  const allOpen = [...data.overdue, ...weekItems];
  const relentlessToday = focus.filter((i) => i.nag_profile === "implacavel").length;

  // Dia exibido a partir da string AAAA-MM-DD do servidor, ao meio-dia UTC:
  // sem deslocar de dia em nenhum fuso do navegador.
  const dayLabel = (date: string, opts: Intl.DateTimeFormatOptions) =>
    new Date(`${date}T12:00:00Z`).toLocaleDateString("pt-BR", { timeZone: "UTC", ...opts });

  // Por Área: contagens de hoje (inclui atrasadas) e da semana.
  const todayIDs = new Set(focus.map((i) => i.id));
  const areas = new Map<string, { name: string; color?: string; today: number; week: number; items: DashboardItem[] }>();
  for (const item of allOpen) {
    const entry = areas.get(item.area_id) ?? {
      name: item.area_name,
      color: item.area_color,
      today: 0,
      week: 0,
      items: [],
    };
    entry.week += 1;
    if (todayIDs.has(item.id)) entry.today += 1;
    entry.items.push(item);
    areas.set(item.area_id, entry);
  }
  const areaList = [...areas.values()].sort((a, b) => b.today - a.today || b.week - a.week);

  return (
    <Layout wide>
      <header className="mb-6">
        <h1 className="text-xl font-semibold capitalize tracking-tight text-slate-100">
          {dayLabel(today.date, { weekday: "long", day: "numeric", month: "long" })}
        </h1>
        <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Stat label="Para hoje" value={focus.length} />
          <Stat label="Atrasadas" value={data.overdue.length} tone={data.overdue.length > 0 ? "danger" : "neutral"} />
          <Stat label="Implacáveis hoje" value={relentlessToday} tone={relentlessToday > 0 ? "danger" : "neutral"} />
          <Stat label="Na semana" value={allOpen.length} />
        </div>
      </header>

      <section className="mb-8">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-slate-100">Foco do dia</h2>
          {focus.length > 0 && (
            <Link to="/hoje" className="text-sm text-indigo-400 hover:underline">
              Reconhecer em Hoje →
            </Link>
          )}
        </div>
        {focus.length === 0 ? (
          <p className="rounded-lg border border-slate-800 bg-slate-900 p-4 text-sm text-slate-400">
            Nada pendente para hoje.
          </p>
        ) : (
          <ul className="space-y-2">
            {focus.map((item) => {
              const overdue = data.overdue.some((o) => o.id === item.id);
              return (
                <li key={item.id} className="rounded-lg border border-slate-800 bg-slate-900 p-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className={`rounded-full border px-2 py-0.5 text-xs ${SEVERITY[item.nag_profile].badge}`}>
                      {SEVERITY[item.nag_profile].label}
                    </span>
                    <span className="font-medium text-slate-100">{item.title}</span>
                    <AreaChip name={item.area_name} color={item.area_color} />
                    <span className={`ml-auto text-sm ${overdue ? "text-red-400" : "text-slate-400"}`}>
                      {overdue
                        ? `atrasada · ${fmt.dayMonth.format(new Date(item.due_at))} ${fmt.time.format(new Date(item.due_at))}`
                        : fmt.time.format(new Date(item.due_at))}
                      {item.current_step > 0 && ` · Degrau ${item.current_step}`}
                    </span>
                  </div>
                  {item.description && <p className="mt-1 text-sm text-slate-400">{item.description}</p>}
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <section className="mb-8">
        <h2 className="mb-3 text-sm font-semibold text-slate-100">Agenda da semana</h2>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 md:grid-cols-4 lg:grid-cols-7">
          {data.days.map((day, i) => {
            const items = [...day.items].sort((a, b) => a.due_at.localeCompare(b.due_at));
            return (
              <div
                key={day.date}
                className={`rounded-lg border p-2 ${
                  i === 0 ? "border-indigo-500 bg-slate-900" : "border-slate-800 bg-slate-900/60"
                }`}
              >
                <p className={`mb-2 text-xs font-medium uppercase ${i === 0 ? "text-indigo-300" : "text-slate-400"}`}>
                  {dayLabel(day.date, { weekday: "short" })}{" "}
                  <span className="text-slate-100">{dayLabel(day.date, { day: "numeric" })}</span>
                </p>
                {items.length === 0 ? (
                  <p className="text-xs text-slate-600">—</p>
                ) : (
                  <ul className="space-y-1">
                    {items.slice(0, MAX_ITEMS_PER_DAY).map((item) => (
                      <li
                        key={item.id}
                        className="border-l-2 pl-2 text-xs"
                        style={{ borderLeftColor: item.area_color || "#475569" }}
                        title={`${item.title} — ${item.area_name}`}
                      >
                        <span className="text-slate-400">{fmt.time.format(new Date(item.due_at))}</span>{" "}
                        <span className="text-slate-200">{item.title}</span>
                      </li>
                    ))}
                    {items.length > MAX_ITEMS_PER_DAY && (
                      <li className="text-xs text-slate-500">+{items.length - MAX_ITEMS_PER_DAY} mais</li>
                    )}
                  </ul>
                )}
              </div>
            );
          })}
        </div>
      </section>

      <section>
        <h2 className="mb-3 text-sm font-semibold text-slate-100">Por Área</h2>
        {areaList.length === 0 ? (
          <p className="rounded-lg border border-slate-800 bg-slate-900 p-4 text-sm text-slate-400">
            Nenhum Compromisso em aberto na semana.{" "}
            <Link to="/compromissos" className="text-indigo-400 hover:underline">
              Criar um
            </Link>
          </p>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {areaList.map((area) => (
              <div key={area.name} className="rounded-lg border border-slate-800 bg-slate-900 p-3">
                <div className="mb-1 flex items-center justify-between">
                  <AreaChip name={area.name} color={area.color} />
                  <span className="text-xs text-slate-400">
                    {area.today} hoje · {area.week} na semana
                  </span>
                </div>
                <ul className="mt-2 space-y-0.5">
                  {[...area.items].sort(bySeverity).slice(0, 3).map((item) => (
                    <li key={item.id} className="truncate text-sm text-slate-300">
                      {item.title}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )}
      </section>
    </Layout>
  );
}

function Stat({ label, value, tone = "neutral" }: { label: string; value: number; tone?: "neutral" | "danger" }) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900 p-3">
      <p className={`text-2xl font-semibold ${tone === "danger" ? "text-red-400" : "text-slate-100"}`}>{value}</p>
      <p className="text-xs text-slate-400">{label}</p>
    </div>
  );
}

function AreaChip({ name, color }: { name: string; color?: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-slate-700 px-2 py-0.5 text-xs text-slate-300">
      <span className="h-2 w-2 rounded-full" style={{ backgroundColor: color || "#94a3b8" }} />
      {name}
    </span>
  );
}
