import { useEffect, useState, type FormEvent } from "react";
import { api } from "../lib/api";
import type { Area } from "../lib/types";

const inputClass = "w-full rounded-md border border-slate-300 px-3 py-2 text-sm";
const buttonClass =
  "rounded-md border border-slate-300 px-3 py-1 text-sm text-slate-700 hover:bg-slate-100 disabled:opacity-50";

type Props = {
  // Compromissos precisa da lista atualizada para o <select> de Área do seu
  // próprio formulário — repassar em vez de cada um buscar /areas por conta
  // própria evita os dois ficarem fora de sincronia depois de um Create.
  onChange?: (areas: Area[]) => void;
};

// Áreas: a divisão de topo da vida do usuário à qual um Compromisso
// pertence (CONTEXT.md) — pré-requisito para criar qualquer Compromisso,
// por isso fica embutido na própria tela de Compromissos, não em Ajustes.
export function AreasSection({ onChange }: Props) {
  const [areas, setAreas] = useState<Area[]>([]);
  const [name, setName] = useState("");
  const [color, setColor] = useState("#64748b");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function reload() {
    const result = await api.get<Area[]>("/areas");
    setAreas(result);
    onChange?.(result);
  }

  useEffect(() => {
    reload().catch(console.error);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await api.post("/areas", { name, color });
      setName("");
      await reload();
    } catch {
      setError("Não foi possível criar a Área — talvez o nome já exista.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="mb-6 space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold text-slate-900">Áreas</h2>

      {areas.length > 0 && (
        <ul className="flex flex-wrap gap-2">
          {areas.map((a) => (
            <li
              key={a.id}
              className="flex items-center gap-1.5 rounded-full border border-slate-300 px-3 py-1 text-sm text-slate-700"
            >
              <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: a.color || "#94a3b8" }} />
              {a.name}
            </li>
          ))}
        </ul>
      )}

      <form onSubmit={handleCreate} className="flex items-end gap-2">
        <div className="flex-1">
          <label htmlFor="area-name" className="block text-sm font-medium text-slate-700">
            Nova Área
          </label>
          <input
            id="area-name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="ex.: Trabalho, Saúde, Pessoal"
            className={inputClass}
          />
        </div>
        <div>
          <label htmlFor="area-color" className="block text-sm font-medium text-slate-700">
            Cor
          </label>
          <input
            id="area-color"
            type="color"
            value={color}
            onChange={(e) => setColor(e.target.value)}
            className="h-[38px] w-14 rounded-md border border-slate-300"
          />
        </div>
        <button type="submit" disabled={submitting} className={buttonClass}>
          Adicionar
        </button>
      </form>

      {error && <p className="text-sm text-red-600">{error}</p>}
    </div>
  );
}
