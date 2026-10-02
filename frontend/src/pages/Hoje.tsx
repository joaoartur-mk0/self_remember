import { useEffect, useState } from "react";
import { Layout } from "../components/Layout";
import { api } from "../lib/api";
import type { Occurrence } from "../lib/types";

// Botões de Reconhecimento com peso visual igual — ARCHITECTURE.md §6 é
// explícito: Concluir, Adiar e Desistir não têm hierarquia entre si.
const ackButtonClass =
  "flex-1 rounded-md border border-slate-700 px-3 py-2 text-sm font-medium text-slate-300 hover:bg-slate-800 disabled:opacity-50";

export function Hoje() {
  const [occurrences, setOccurrences] = useState<Occurrence[] | null>(null);
  const [snoozingID, setSnoozingID] = useState<string | null>(null);
  const [snoozeValue, setSnoozeValue] = useState("");
  const [busyID, setBusyID] = useState<string | null>(null);

  async function reload() {
    const data = await api.get<Occurrence[]>("/occurrences/today");
    setOccurrences(data);
  }

  useEffect(() => {
    reload().catch(() => setOccurrences([]));
  }, []);

  async function act(id: string, action: "complete" | "dismiss") {
    setBusyID(id);
    try {
      await api.post(`/occurrences/${id}/${action}`);
      await reload();
    } finally {
      setBusyID(null);
    }
  }

  async function confirmSnooze(id: string) {
    if (!snoozeValue) return;
    setBusyID(id);
    try {
      await api.post(`/occurrences/${id}/snooze`, {
        due_at: new Date(snoozeValue).toISOString(),
      });
      setSnoozingID(null);
      setSnoozeValue("");
      await reload();
    } finally {
      setBusyID(null);
    }
  }

  if (occurrences === null) {
    return (
      <Layout>
        <p className="text-slate-400">Carregando...</p>
      </Layout>
    );
  }

  return (
    <Layout>
      <h1 className="mb-4 text-lg font-semibold text-slate-100">Hoje</h1>

      {occurrences.length === 0 && <p className="text-slate-400">Nada vencendo hoje.</p>}

      <ul className="space-y-3">
        {occurrences.map((occ) => (
          <li key={occ.id} className="rounded-lg border border-slate-800 bg-slate-900 p-4">
            <div className="mb-3">
              <p className="font-medium text-slate-100">{occ.title}</p>
              <p className="text-sm text-slate-400">
                Vence em {new Date(occ.due_at).toLocaleString()} · Degrau {occ.current_step}
              </p>
            </div>

            {snoozingID === occ.id ? (
              <div className="flex gap-2">
                <input
                  type="datetime-local"
                  value={snoozeValue}
                  onChange={(e) => setSnoozeValue(e.target.value)}
                  className="flex-1 rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-sm text-slate-100 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                />
                <button
                  type="button"
                  onClick={() => confirmSnooze(occ.id)}
                  disabled={busyID === occ.id}
                  className={ackButtonClass}
                >
                  Confirmar
                </button>
                <button
                  type="button"
                  onClick={() => setSnoozingID(null)}
                  className={ackButtonClass}
                >
                  Cancelar
                </button>
              </div>
            ) : (
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => act(occ.id, "complete")}
                  disabled={busyID === occ.id}
                  className={ackButtonClass}
                >
                  Concluir
                </button>
                <button
                  type="button"
                  onClick={() => setSnoozingID(occ.id)}
                  disabled={busyID === occ.id}
                  className={ackButtonClass}
                >
                  Adiar
                </button>
                <button
                  type="button"
                  onClick={() => act(occ.id, "dismiss")}
                  disabled={busyID === occ.id}
                  className={ackButtonClass}
                >
                  Desistir
                </button>
              </div>
            )}
          </li>
        ))}
      </ul>
    </Layout>
  );
}
