import { useEffect, useState } from "react";
import { Link } from "react-router";
import { Layout } from "../components/Layout";
import { api } from "../lib/api";
import { describeRecurrence } from "../lib/recurrence";
import type { Area, Commitment, NagProfile } from "../lib/types";

const PROFILE_LABEL: Record<NagProfile, string> = {
  leve: "Leve",
  normal: "Normal",
  implacavel: "Implacável",
};

// Lista de todos os Compromissos ativos. Criar um novo abre o formulário em
// /compromissos/novo (NovoCompromisso).
export function Compromissos() {
  const [commitments, setCommitments] = useState<Commitment[] | null>(null);
  const [areas, setAreas] = useState<Map<string, Area>>(new Map());

  async function reload() {
    setCommitments(await api.get<Commitment[]>("/commitments", { include_archived: false }));
  }

  useEffect(() => {
    reload().catch(() => setCommitments([]));
    api
      .get<Area[]>("/areas")
      .then((list) => setAreas(new Map(list.map((a) => [a.id, a]))))
      .catch(console.error);
  }, []);

  async function archive(id: string) {
    await api.post(`/commitments/${id}/archive`);
    await reload();
  }

  return (
    <Layout>
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-lg font-semibold text-slate-100">Compromissos</h1>
        <Link
          to="/compromissos/novo"
          className="rounded-md bg-indigo-500 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-400"
        >
          Novo compromisso
        </Link>
      </div>

      {commitments === null && <p className="text-slate-400">Carregando...</p>}

      {commitments?.length === 0 && (
        <p className="rounded-lg border border-slate-800 bg-slate-900 p-4 text-sm text-slate-400">
          Nenhum compromisso ainda. Use “Novo compromisso” para criar o primeiro.
        </p>
      )}

      <ul className="space-y-2">
        {commitments?.map((c) => {
          const area = areas.get(c.area_id);
          return (
            <li
              key={c.id}
              className="flex items-start justify-between gap-3 rounded-lg border border-slate-800 bg-slate-900 p-3"
            >
              <div className="min-w-0">
                <p className="text-slate-100">{c.title}</p>
                <p className="text-sm text-slate-400">{c.description}</p>
                <p className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500">
                  {area && (
                    <span className="inline-flex items-center gap-1.5">
                      <span
                        className="h-2 w-2 rounded-full"
                        style={{ backgroundColor: area.color || "#94a3b8" }}
                      />
                      {area.name}
                    </span>
                  )}
                  <span>{describeRecurrence(c)}</span>
                  <span>{PROFILE_LABEL[c.nag_profile]}</span>
                </p>
              </div>
              <button
                type="button"
                onClick={() => archive(c.id)}
                className="shrink-0 rounded-md border border-slate-700 px-3 py-1 text-sm text-slate-300 hover:bg-slate-800"
              >
                Arquivar
              </button>
            </li>
          );
        })}
      </ul>
    </Layout>
  );
}
