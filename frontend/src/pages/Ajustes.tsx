import { useEffect, useState, type FormEvent } from "react";
import { ChannelsSection } from "../components/ChannelsSection";
import { Layout } from "../components/Layout";
import { WhatsAppInstancePanel } from "../components/WhatsAppInstancePanel";
import { api, ApiError } from "../lib/api";
import type { CarryOverPolicy, NagProfile, Settings } from "../lib/types";

const inputClass = "w-full rounded-md border border-slate-300 px-3 py-2 text-sm";

// Lista completa de fusos IANA que o runtime conhece — evita que o fuso
// gravado no backend (ex.: via cmd/admin create-user) não tenha <option>
// correspondente e o <select> caia num valor diferente do gravado.
const TIMEZONES = Intl.supportedValuesOf("timeZone");

export function Ajustes() {
  const [timezone, setTimezone] = useState("America/Sao_Paulo");
  const [quietStart, setQuietStart] = useState("");
  const [quietEnd, setQuietEnd] = useState("");
  const [defaultNagProfile, setDefaultNagProfile] = useState<NagProfile>("normal");
  const [defaultCarryOverPolicy, setDefaultCarryOverPolicy] = useState<CarryOverPolicy>("stack");
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
        setDefaultNagProfile(s.default_nag_profile);
        setDefaultCarryOverPolicy(s.default_carry_over_policy);
      })
      .catch((err) => {
        // 404: usuário ainda sem Ajustes gravados — segue com os padrões do
        // formulário em vez de bloquear a tela. Qualquer outro erro só é
        // registrado: não há mais nada a fazer no load da tela.
        if (!(err instanceof ApiError && err.status === 404)) console.error(err);
      });
  }, []);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSaved(false);

    if (Boolean(quietStart) !== Boolean(quietEnd)) {
      setError("Informe os dois horários da Janela de Silêncio, ou deixe os dois em branco.");
      return;
    }

    setSubmitting(true);
    try {
      await api.put("/settings", {
        timezone,
        quiet_hours_start: quietStart || null,
        quiet_hours_end: quietEnd || null,
        default_nag_profile: defaultNagProfile,
        default_carry_over_policy: defaultCarryOverPolicy,
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
          <label htmlFor="timezone" className="block text-sm font-medium text-slate-700">
            Fuso horário
          </label>
          <select
            id="timezone"
            value={timezone}
            onChange={(e) => setTimezone(e.target.value)}
            className={inputClass}
          >
            {TIMEZONES.map((tz) => (
              <option key={tz} value={tz}>
                {tz}
              </option>
            ))}
          </select>
        </div>

        <div>
          <span className="block text-sm font-medium text-slate-700">Janela de Silêncio</span>
          <div className="flex items-center gap-2">
            <label htmlFor="quiet-start" className="sr-only">
              Início da Janela de Silêncio
            </label>
            <input
              id="quiet-start"
              type="time"
              value={quietStart}
              onChange={(e) => setQuietStart(e.target.value)}
              className={inputClass}
            />
            <span className="text-slate-500">até</span>
            <label htmlFor="quiet-end" className="sr-only">
              Fim da Janela de Silêncio
            </label>
            <input
              id="quiet-end"
              type="time"
              value={quietEnd}
              onChange={(e) => setQuietEnd(e.target.value)}
              className={inputClass}
            />
          </div>
          <p className="mt-1 text-xs text-slate-500">Deixe os dois em branco para desativar.</p>
        </div>

        <div className="flex gap-4">
          <div className="flex-1">
            <label htmlFor="default-nag-profile" className="block text-sm font-medium text-slate-700">
              Perfil de Insistência padrão
            </label>
            <select
              id="default-nag-profile"
              value={defaultNagProfile}
              onChange={(e) => setDefaultNagProfile(e.target.value as NagProfile)}
              className={inputClass}
            >
              <option value="leve">Leve</option>
              <option value="normal">Normal</option>
              <option value="implacavel">Implacável</option>
            </select>
          </div>
          <div className="flex-1">
            <label htmlFor="default-carry-over-policy" className="block text-sm font-medium text-slate-700">
              Política de Acúmulo padrão
            </label>
            <select
              id="default-carry-over-policy"
              value={defaultCarryOverPolicy}
              onChange={(e) => setDefaultCarryOverPolicy(e.target.value as CarryOverPolicy)}
              className={inputClass}
            >
              <option value="stack">Acumular</option>
              <option value="replace">Substituir</option>
            </select>
          </div>
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

      <div className="mt-6 space-y-6">
        <ChannelsSection />
        <WhatsAppInstancePanel />
      </div>
    </Layout>
  );
}
