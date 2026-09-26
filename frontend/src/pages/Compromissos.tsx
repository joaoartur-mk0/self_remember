import { useEffect, useState, type FormEvent } from "react";
import { Layout } from "../components/Layout";
import { api } from "../lib/api";
import type { Area, Commitment, Label } from "../lib/types";

const inputClass = "w-full rounded-md border border-slate-300 px-3 py-2 text-sm";

export function Compromissos() {
  const [commitments, setCommitments] = useState<Commitment[]>([]);
  const [areas, setAreas] = useState<Area[]>([]);
  const [labels, setLabels] = useState<Label[]>([]);

  const [title, setTitle] = useState("");
  const [areaID, setAreaID] = useState("");
  const [recurrenceStart, setRecurrenceStart] = useState("");
  const [recurrenceRule, setRecurrenceRule] = useState("");
  const [nagProfile, setNagProfile] = useState("normal");
  const [carryOverPolicy, setCarryOverPolicy] = useState("stack");
  const [labelIDs, setLabelIDs] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function reloadCommitments() {
    setCommitments(await api.get<Commitment[]>("/commitments", { include_archived: false }));
  }

  useEffect(() => {
    reloadCommitments();
    api.get<Area[]>("/areas").then(setAreas);
    api.get<Label[]>("/labels").then(setLabels);
  }, []);

  function toggleLabel(id: string) {
    setLabelIDs((prev) => (prev.includes(id) ? prev.filter((l) => l !== id) : [...prev, id]));
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await api.post("/commitments", {
        title,
        area_id: areaID,
        recurrence_start: new Date(recurrenceStart).toISOString(),
        recurrence_rule: recurrenceRule || undefined,
        nag_profile: nagProfile,
        carry_over_policy: carryOverPolicy,
        label_ids: labelIDs,
      });
      setTitle("");
      setRecurrenceStart("");
      setRecurrenceRule("");
      setLabelIDs([]);
      await reloadCommitments();
    } catch {
      setError("Não foi possível criar o Compromisso — confira os campos.");
    } finally {
      setSubmitting(false);
    }
  }

  async function archive(id: string) {
    await api.post(`/commitments/${id}/archive`);
    await reloadCommitments();
  }

  return (
    <Layout>
      <h1 className="mb-4 text-lg font-semibold text-slate-900">Compromissos</h1>

      <form onSubmit={handleSubmit} className="mb-8 space-y-3 rounded-lg border border-slate-200 bg-white p-4">
        <div>
          <label className="block text-sm font-medium text-slate-700">Título</label>
          <input required value={title} onChange={(e) => setTitle(e.target.value)} className={inputClass} />
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700">Área</label>
          <select required value={areaID} onChange={(e) => setAreaID(e.target.value)} className={inputClass}>
            <option value="" disabled>
              Selecione
            </option>
            {areas.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700">Vencimento inicial</label>
          <input
            required
            type="datetime-local"
            value={recurrenceStart}
            onChange={(e) => setRecurrenceStart(e.target.value)}
            className={inputClass}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700">
            Regra de recorrência (RRULE, opcional)
          </label>
          <input
            value={recurrenceRule}
            onChange={(e) => setRecurrenceRule(e.target.value)}
            placeholder="FREQ=WEEKLY;BYDAY=MO"
            className={inputClass}
          />
        </div>

        <div className="flex gap-4">
          <div className="flex-1">
            <label className="block text-sm font-medium text-slate-700">Perfil de Insistência</label>
            <select value={nagProfile} onChange={(e) => setNagProfile(e.target.value)} className={inputClass}>
              <option value="leve">Leve</option>
              <option value="normal">Normal</option>
              <option value="implacavel">Implacável</option>
            </select>
          </div>
          <div className="flex-1">
            <label className="block text-sm font-medium text-slate-700">Política de Acúmulo</label>
            <select
              value={carryOverPolicy}
              onChange={(e) => setCarryOverPolicy(e.target.value)}
              className={inputClass}
            >
              <option value="stack">Acumular</option>
              <option value="replace">Substituir</option>
            </select>
          </div>
        </div>

        {labels.length > 0 && (
          <div>
            <label className="block text-sm font-medium text-slate-700">Etiquetas</label>
            <div className="flex flex-wrap gap-2">
              {labels.map((l) => (
                <button
                  type="button"
                  key={l.id}
                  onClick={() => toggleLabel(l.id)}
                  className={`rounded-full border px-3 py-1 text-sm ${
                    labelIDs.includes(l.id)
                      ? "border-slate-900 bg-slate-900 text-white"
                      : "border-slate-300 text-slate-700"
                  }`}
                >
                  {l.name}
                </button>
              ))}
            </div>
          </div>
        )}

        {error && <p className="text-sm text-red-600">{error}</p>}

        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          Criar Compromisso
        </button>
      </form>

      <ul className="space-y-2">
        {commitments.map((c) => (
          <li
            key={c.id}
            className="flex items-center justify-between rounded-lg border border-slate-200 bg-white p-3"
          >
            <span className="text-slate-900">{c.title}</span>
            <button
              type="button"
              onClick={() => archive(c.id)}
              className="rounded-md border border-slate-300 px-3 py-1 text-sm text-slate-700 hover:bg-slate-100"
            >
              Arquivar
            </button>
          </li>
        ))}
      </ul>
    </Layout>
  );
}
