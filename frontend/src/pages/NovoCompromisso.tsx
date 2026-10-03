import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { Layout } from "../components/Layout";
import { api, ApiError } from "../lib/api";
import { WEEKDAYS } from "../lib/recurrence";
import type { Area, CarryOverPolicy, Label, NagProfile, Settings } from "../lib/types";

const inputClass =
  "w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500";
const pillClass = (active: boolean) =>
  `rounded-full border px-3 py-1 text-sm ${
    active ? "border-indigo-500 bg-indigo-500 text-white" : "border-slate-700 text-slate-300"
  }`;

type RecurrenceKind = "pontual" | "diario" | "semanal" | "mensal";

const RECURRENCE_KINDS: { value: RecurrenceKind; label: string }[] = [
  { value: "pontual", label: "Pontual" },
  { value: "diario", label: "Diário" },
  { value: "semanal", label: "Semanal" },
  { value: "mensal", label: "Mensal" },
];

export function NovoCompromisso() {
  const navigate = useNavigate();
  const [areas, setAreas] = useState<Area[]>([]);
  const [labels, setLabels] = useState<Label[]>([]);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [areaID, setAreaID] = useState("");
  const [recurrenceKind, setRecurrenceKind] = useState<RecurrenceKind>("pontual");
  const [dueDate, setDueDate] = useState(""); // só Pontual: datetime-local
  const [dueTime, setDueTime] = useState(""); // Diário/Semanal/Mensal: só a hora
  const [weekdays, setWeekdays] = useState<string[]>([]); // só Semanal
  const [monthDay, setMonthDay] = useState(1); // só Mensal
  const [nagProfile, setNagProfile] = useState<NagProfile>("normal");
  const [carryOverPolicy, setCarryOverPolicy] = useState<CarryOverPolicy>("stack");
  const [labelIDs, setLabelIDs] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    api.get<Area[]>("/areas").then(setAreas).catch(console.error);
    api.get<Label[]>("/labels").then(setLabels).catch(console.error);
    // Pré-seleciona os padrões de Ajustes — 404 (sem Ajustes gravados ainda)
    // apenas mantém "normal"/"stack", os padrões locais destes campos.
    api
      .get<Settings>("/settings")
      .then((s) => {
        setNagProfile(s.default_nag_profile);
        setCarryOverPolicy(s.default_carry_over_policy);
      })
      .catch((err) => {
        if (!(err instanceof ApiError && err.status === 404)) console.error(err);
      });
  }, []);

  function toggleLabel(id: string) {
    setLabelIDs((prev) => (prev.includes(id) ? prev.filter((l) => l !== id) : [...prev, id]));
  }

  function toggleWeekday(value: string) {
    setWeekdays((prev) => (prev.includes(value) ? prev.filter((w) => w !== value) : [...prev, value]));
  }

  // O backend reinterpreta os DÍGITOS de recurrence_start como hora de
  // parede no fuso configurado em Ajustes (wallClockIn, ARCHITECTURE.md
  // §3.1) — o deslocamento que a string realmente carrega é ignorado. Por
  // isso NUNCA construímos um Date e chamamos toISOString() aqui: isso faz
  // o navegador converter pelo fuso DELE, corrompendo os dígitos toda vez
  // que o fuso do navegador não é UTC+0 (e, se o desvio empurrar a hora pro
  // passado, o Compromisso pontual nunca materializa Ocorrência nenhuma —
  // NextOccurrenceAfter devolve ok=false silenciosamente). Os dígitos
  // literais que o usuário escolheu, com sufixo "Z", é o que o backend
  // espera — nunca uma conversão de verdade.
  function toRecurrenceStart(year: number, month: number, day: number, hours: number, minutes: number): string {
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${year}-${pad(month)}-${pad(day)}T${pad(hours)}:${pad(minutes)}:00.000Z`;
  }

  // Monta recurrence_start/recurrence_rule a partir dos campos específicos de
  // cada tipo de recorrência. null significa "campos insuficientes" — quem
  // chama decide a mensagem de erro, não esta função. A data de
  // recurrence_start fora do modo Pontual é só uma âncora (DTSTART) para o
  // rrule-go contar os períodos a partir dela; quem decide em quais dias a
  // recorrência cai de verdade é BYDAY/BYMONTHDAY, por isso usar "hoje" aqui
  // é seguro mesmo que hoje não seja um desses dias.
  function buildRecurrence(): { recurrenceStart: string; recurrenceRule?: string } | null {
    if (recurrenceKind === "pontual") {
      // dueDate já vem de um <input type="datetime-local"> como
      // "AAAA-MM-DDTHH:mm" — exatamente os dígitos que queremos, sem
      // nenhuma interpretação de fuso no meio.
      if (!dueDate) return null;
      return { recurrenceStart: `${dueDate}:00.000Z` };
    }

    if (!dueTime) return null;
    const [hours, minutes] = dueTime.split(":").map(Number);
    const today = new Date();
    const recurrenceStart = toRecurrenceStart(today.getFullYear(), today.getMonth() + 1, today.getDate(), hours, minutes);

    if (recurrenceKind === "diario") {
      return { recurrenceStart, recurrenceRule: "FREQ=DAILY" };
    }
    if (recurrenceKind === "semanal") {
      if (weekdays.length === 0) return null;
      return { recurrenceStart, recurrenceRule: `FREQ=WEEKLY;BYDAY=${weekdays.join(",")}` };
    }
    return { recurrenceStart, recurrenceRule: `FREQ=MONTHLY;BYMONTHDAY=${monthDay}` };
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);

    const recurrence = buildRecurrence();
    if (!recurrence) {
      setError(
        recurrenceKind === "semanal"
          ? "Selecione ao menos um dia da semana."
          : "Informe a data/horário da recorrência.",
      );
      return;
    }

    setSubmitting(true);
    try {
      await api.post("/commitments", {
        title,
        description,
        area_id: areaID,
        recurrence_start: recurrence.recurrenceStart,
        recurrence_rule: recurrence.recurrenceRule,
        nag_profile: nagProfile,
        carry_over_policy: carryOverPolicy,
        label_ids: labelIDs,
      });
      navigate("/compromissos");
    } catch {
      setError("Não foi possível criar o Compromisso — confira os campos.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Layout>
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-lg font-semibold text-slate-100">Novo compromisso</h1>
        <Link to="/compromissos" className="text-sm text-slate-400 hover:text-slate-100">
          ← Voltar à lista
        </Link>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3 rounded-lg border border-slate-800 bg-slate-900 p-4">
        <div>
          <label htmlFor="title" className="block text-sm font-medium text-slate-300">
            Título
          </label>
          <input
            id="title"
            required
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className={inputClass}
          />
        </div>

        <div>
          <label htmlFor="description" className="block text-sm font-medium text-slate-300">
            Descrição
          </label>
          <textarea
            id="description"
            required
            rows={2}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="O que é, pra lembrar na hora da Cobrança — ex.: Losartana, 1 comprimido, com água"
            className={inputClass}
          />
        </div>

        <div>
          <label htmlFor="area" className="block text-sm font-medium text-slate-300">
            Área
          </label>
          <select
            id="area"
            required
            value={areaID}
            onChange={(e) => setAreaID(e.target.value)}
            className={inputClass}
          >
            <option value="" disabled>
              Selecione
            </option>
            {areas.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
          {areas.length === 0 && (
            <p className="mt-1 text-xs text-slate-400">
              Nenhuma Área ainda.{" "}
              <Link to="/cadastros/areas" className="text-indigo-400 hover:underline">
                Cadastre uma
              </Link>{" "}
              para criar o compromisso.
            </p>
          )}
        </div>

        <div>
          <span className="block text-sm font-medium text-slate-300">Recorrência</span>
          <div className="flex flex-wrap gap-2">
            {RECURRENCE_KINDS.map((k) => (
              <button
                type="button"
                key={k.value}
                onClick={() => setRecurrenceKind(k.value)}
                className={pillClass(recurrenceKind === k.value)}
              >
                {k.label}
              </button>
            ))}
          </div>
        </div>

        {recurrenceKind === "pontual" && (
          <div>
            <label htmlFor="due-date" className="block text-sm font-medium text-slate-300">
              Data e hora
            </label>
            <input
              id="due-date"
              required
              type="datetime-local"
              value={dueDate}
              onChange={(e) => setDueDate(e.target.value)}
              className={inputClass}
            />
          </div>
        )}

        {recurrenceKind === "diario" && (
          <div>
            <label htmlFor="due-time" className="block text-sm font-medium text-slate-300">
              Horário
            </label>
            <input
              id="due-time"
              required
              type="time"
              value={dueTime}
              onChange={(e) => setDueTime(e.target.value)}
              className={inputClass}
            />
          </div>
        )}

        {recurrenceKind === "semanal" && (
          <div className="space-y-2">
            <div>
              <span className="block text-sm font-medium text-slate-300">Dias da semana</span>
              <div className="flex flex-wrap gap-2">
                {WEEKDAYS.map((w) => (
                  <button
                    type="button"
                    key={w.value}
                    onClick={() => toggleWeekday(w.value)}
                    className={pillClass(weekdays.includes(w.value))}
                  >
                    {w.label}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <label htmlFor="due-time" className="block text-sm font-medium text-slate-300">
                Horário
              </label>
              <input
                id="due-time"
                required
                type="time"
                value={dueTime}
                onChange={(e) => setDueTime(e.target.value)}
                className={inputClass}
              />
            </div>
          </div>
        )}

        {recurrenceKind === "mensal" && (
          <div className="flex gap-4">
            <div>
              <label htmlFor="month-day" className="block text-sm font-medium text-slate-300">
                Dia do mês
              </label>
              <input
                id="month-day"
                required
                type="number"
                min={1}
                max={31}
                value={monthDay}
                onChange={(e) => setMonthDay(Number(e.target.value))}
                className={`${inputClass} w-20`}
              />
            </div>
            <div className="flex-1">
              <label htmlFor="due-time" className="block text-sm font-medium text-slate-300">
                Horário
              </label>
              <input
                id="due-time"
                required
                type="time"
                value={dueTime}
                onChange={(e) => setDueTime(e.target.value)}
                className={inputClass}
              />
            </div>
          </div>
        )}

        <div className="flex gap-4">
          <div className="flex-1">
            <label htmlFor="nag-profile" className="block text-sm font-medium text-slate-300">
              Perfil de Insistência
            </label>
            <select
              id="nag-profile"
              value={nagProfile}
              onChange={(e) => setNagProfile(e.target.value as NagProfile)}
              className={inputClass}
            >
              <option value="leve">Leve</option>
              <option value="normal">Normal</option>
              <option value="implacavel">Implacável</option>
            </select>
          </div>
          <div className="flex-1">
            <label htmlFor="carry-over-policy" className="block text-sm font-medium text-slate-300">
              Política de Acúmulo
            </label>
            <select
              id="carry-over-policy"
              value={carryOverPolicy}
              onChange={(e) => setCarryOverPolicy(e.target.value as CarryOverPolicy)}
              className={inputClass}
            >
              <option value="stack">Acumular</option>
              <option value="replace">Substituir</option>
            </select>
          </div>
        </div>

        {labels.length > 0 && (
          <div>
            <span className="block text-sm font-medium text-slate-300">Etiquetas</span>
            <div className="flex flex-wrap gap-2">
              {labels.map((l) => (
                <button
                  type="button"
                  key={l.id}
                  onClick={() => toggleLabel(l.id)}
                  className={pillClass(labelIDs.includes(l.id))}
                >
                  {l.name}
                </button>
              ))}
            </div>
          </div>
        )}

        {error && <p className="text-sm text-red-400">{error}</p>}

        <button
          type="submit"
          disabled={submitting}
          className="rounded-md bg-indigo-500 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-400 disabled:opacity-50"
        >
          Criar Compromisso
        </button>
      </form>

    </Layout>
  );
}
