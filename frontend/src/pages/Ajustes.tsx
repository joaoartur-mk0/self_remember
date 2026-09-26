import { useEffect, useState, type FormEvent } from "react";
import { Layout } from "../components/Layout";
import { api, ApiError } from "../lib/api";
import type { Settings } from "../lib/types";

const inputClass = "w-full rounded-md border border-slate-300 px-3 py-2 text-sm";

// Lista curta e comum de fusos IANA — um seletor exaustivo (Intl.supportedValuesOf)
// existe, mas cobrir a lista inteira é trabalho de UI própria, fora deste corte.
const TIMEZONES = [
  "America/Sao_Paulo",
  "America/Manaus",
  "America/Noronha",
  "America/New_York",
  "America/Los_Angeles",
  "Europe/Lisbon",
  "Europe/London",
  "UTC",
];

export function Ajustes() {
  const [timezone, setTimezone] = useState("America/Sao_Paulo");
  const [quietStart, setQuietStart] = useState("");
  const [quietEnd, setQuietEnd] = useState("");
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    api
      .get<Settings>("/settings")
      .then((s) => {
        setTimezone(s.timezone);
        setQuietStart(s.quiet_hours_start ?? "");
        setQuietEnd(s.quiet_hours_end ?? "");
      })
      .catch((err) => {
        // 404: usuário ainda sem Ajustes gravados — segue com os padrões do
        // formulário em vez de bloquear a tela.
        if (!(err instanceof ApiError && err.status === 404)) throw err;
      });
  }, []);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSaved(false);
    setSubmitting(true);
    try {
      await api.put("/settings", {
        timezone,
        quiet_hours_start: quietStart || null,
        quiet_hours_end: quietEnd || null,
      });
      setSaved(true);
    } catch {
      setError("Não foi possível gravar os Ajustes — confira o fuso e os horários.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Layout>
      <h1 className="mb-4 text-lg font-semibold text-slate-900">Ajustes</h1>

      <form onSubmit={handleSubmit} className="max-w-sm space-y-4 rounded-lg border border-slate-200 bg-white p-4">
        <div>
          <label className="block text-sm font-medium text-slate-700">Fuso horário</label>
          <select value={timezone} onChange={(e) => setTimezone(e.target.value)} className={inputClass}>
            {TIMEZONES.map((tz) => (
              <option key={tz} value={tz}>
                {tz}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-slate-700">Janela de Silêncio</label>
          <div className="flex items-center gap-2">
            <input
              type="time"
              value={quietStart}
              onChange={(e) => setQuietStart(e.target.value)}
              className={inputClass}
            />
            <span className="text-slate-500">até</span>
            <input
              type="time"
              value={quietEnd}
              onChange={(e) => setQuietEnd(e.target.value)}
              className={inputClass}
            />
          </div>
          <p className="mt-1 text-xs text-slate-500">Deixe os dois em branco para desativar.</p>
        </div>

        {error && <p className="text-sm text-red-600">{error}</p>}
        {saved && <p className="text-sm text-green-600">Ajustes gravados.</p>}

        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          Salvar
        </button>
      </form>
    </Layout>
  );
}
