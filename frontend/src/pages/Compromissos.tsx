import { useEffect, useState, type FormEvent } from "react";
import { AreasSection } from "../components/AreasSection";
import { Layout } from "../components/Layout";
import { api, ApiError } from "../lib/api";
import type { Area, CarryOverPolicy, Commitment, Label, NagProfile, Settings } from "../lib/types";

const inputClass = "w-full rounded-md border border-slate-300 px-3 py-2 text-sm";
const pillClass = (active: boolean) =>
  `rounded-full border px-3 py-1 text-sm ${
    active ? "border-slate-900 bg-slate-900 text-white" : "border-slate-300 text-slate-700"
  }`;

type RecurrenceKind = "pontual" | "diario" | "semanal" | "mensal";

const RECURRENCE_KINDS: { value: RecurrenceKind; label: string }[] = [
  { value: "pontual", label: "Pontual" },
  { value: "diario", label: "Diário" },
  { value: "semanal", label: "Semanal" },
  { value: "mensal", label: "Mensal" },
];

// Valores no formato BYDAY do RRULE (RFC 5545) — a expansão real
// (backend/internal/domain/commitment/recurrence.go, via rrule-go) aceita
// qualquer RRULE válida, então BYDAY/BYMONTHDAY explícitos aqui bastam; o
// backend não precisa de nenhuma mudança.
const WEEKDAYS: { value: string; label: string }[] = [
  { value: "MO", label: "Seg" },
  { value: "TU", label: "Ter" },
  { value: "WE", label: "Qua" },
  { value: "TH", label: "Qui" },
  { value: "FR", label: "Sex" },
  { value: "SA", label: "Sáb" },
  { value: "SU", label: "Dom" },
];

export function Compromissos() {
  const [commitments, setCommitments] = useState<Commitment[]>([]);
  const [areas, setAreas] = useState<Area[]>([]);
  const [labels, setLabels] = useState<Label[]>([]);

  const [title, setTitle] = useState("");
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

  async function reloadCommitments() {
    setCommitments(await api.get<Commitment[]>("/commitments", { include_archived: false }));
  }

  useEffect(() => {
    reloadCommitments().catch(console.error);
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

  // Monta recurrence_start/recurrence_rule a partir dos campos específicos de
  // cada tipo de recorrência. null significa "campos insuficientes" — quem
  // chama decide a mensagem de erro, não esta função. A data de
  // recurrence_start fora do modo Pontual é só uma âncora (DTSTART) para o
  // rrule-go contar os períodos a partir dela; quem decide em quais dias a
  // recorrência cai de verdade é BYDAY/BYMONTHDAY, por isso usar "hoje" aqui
  // é seguro mesmo que hoje não seja um desses dias.
  function buildRecurrence(): { recurrenceStart: string; recurrenceRule?: string } | null {
    if (recurrenceKind === "pontual") {
      if (!dueDate) return null;
      return { recurrenceStart: new Date(dueDate).toISOString() };
    }

    if (!dueTime) return null;
    const [hours, minutes] = dueTime.split(":").map(Number);
    const now = new Date();
    const start = new Date(now.getFullYear(), now.getMonth(), now.getDate(), hours, minutes);

    if (recurrenceKind === "diario") {
      return { recurrenceStart: start.toISOString(), recurrenceRule: "FREQ=DAILY" };
    }
    if (recurrenceKind === "semanal") {
      if (weekdays.length === 0) return null;
      return { recurrenceStart: start.toISOString(), recurrenceRule: `FREQ=WEEKLY;BYDAY=${weekdays.join(",")}` };
    }
    return { recurrenceStart: start.toISOString(), recurrenceRule: `FREQ=MONTHLY;BYMONTHDAY=${monthDay}` };
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
        area_id: areaID,
        recurrence_start: recurrence.recurrenceStart,
        recurrence_rule: recurrence.recurrenceRule,
        nag_profile: nagProfile,
        carry_over_policy: carryOverPolicy,
        label_ids: labelIDs,
      });
      setTitle("");
      setRecurrenceKind("pontual");
      setDueDate("");
      setDueTime("");
      setWeekdays([]);
      setMonthDay(1);
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

      <AreasSection onChange={setAreas} />

      <form onSubmit={handleSubmit} className="mb-8 space-y-3 rounded-lg border border-slate-200 bg-white p-4">
        <div>
          <label htmlFor="title" className="block text-sm font-medium text-slate-700">
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
          <label htmlFor="area" className="block text-sm font-medium text-slate-700">
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
        </div>

        <div>
          <span className="block text-sm font-medium text-slate-700">Recorrência</span>
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
            <label htmlFor="due-date" className="block text-sm font-medium text-slate-700">
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
            <label htmlFor="due-time" className="block text-sm font-medium text-slate-700">
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
              <span className="block text-sm font-medium text-slate-700">Dias da semana</span>
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
              <label htmlFor="due-time" className="block text-sm font-medium text-slate-700">
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
              <label htmlFor="month-day" className="block text-sm font-medium text-slate-700">
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
              <label htmlFor="due-time" className="block text-sm font-medium text-slate-700">
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
            <label htmlFor="nag-profile" className="block text-sm font-medium text-slate-700">
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
            <label htmlFor="carry-over-policy" className="block text-sm font-medium text-slate-700">
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
            <span className="block text-sm font-medium text-slate-700">Etiquetas</span>
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
