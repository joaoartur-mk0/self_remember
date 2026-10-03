import type { Commitment } from "./types";

// Valores no formato BYDAY do RRULE (RFC 5545) — a expansão real
// (backend/internal/domain/commitment/recurrence.go, via rrule-go) aceita
// qualquer RRULE válida, então BYDAY/BYMONTHDAY explícitos bastam.
export const WEEKDAYS: { value: string; label: string }[] = [
  { value: "MO", label: "Seg" },
  { value: "TU", label: "Ter" },
  { value: "WE", label: "Qua" },
  { value: "TH", label: "Qui" },
  { value: "FR", label: "Sex" },
  { value: "SA", label: "Sáb" },
  { value: "SU", label: "Dom" },
];

// Resume a recorrência de um Compromisso para a lista. Data e hora saem dos
// DÍGITOS de recurrence_start, sem passar por Date: o backend os trata como
// hora de parede no fuso de Ajustes (wallClockIn), então converter pelo fuso
// do navegador mostraria outro horário.
export function describeRecurrence(c: Pick<Commitment, "recurrence_start" | "recurrence_rule">): string {
  const time = c.recurrence_start.slice(11, 16);
  const rule = c.recurrence_rule ?? "";

  if (!rule) {
    const [year, month, day] = c.recurrence_start.slice(0, 10).split("-");
    return `${day}/${month}/${year} às ${time}`;
  }

  const params: Record<string, string | undefined> = {};
  for (const part of rule.split(";")) {
    const [key, value] = part.split("=");
    params[key] = value;
  }
  switch (params.FREQ) {
    case "DAILY":
      return `Todo dia às ${time}`;
    case "WEEKLY": {
      const days = (params.BYDAY ?? "")
        .split(",")
        .map((code: string) => WEEKDAYS.find((w) => w.value === code)?.label ?? code)
        .join(", ");
      return days ? `${days} às ${time}` : `Toda semana às ${time}`;
    }
    case "MONTHLY":
      return params.BYMONTHDAY
        ? `Dia ${params.BYMONTHDAY} de cada mês às ${time}`
        : `Todo mês às ${time}`;
    default:
      return rule;
  }
}
