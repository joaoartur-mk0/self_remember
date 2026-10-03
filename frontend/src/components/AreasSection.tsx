import { useEffect, useState, type FormEvent } from "react";
import { api, ApiError } from "../lib/api";
import type { Area } from "../lib/types";

const inputClass =
  "w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500";
const buttonClass =
  "rounded-md border border-slate-700 px-3 py-1 text-sm text-slate-300 hover:bg-slate-800 disabled:opacity-50";

type Props = {
  // Compromissos precisa da lista atualizada para o <select> de Área do seu
  // próprio formulário — repassar em vez de cada um buscar /areas por conta
  // própria evita os dois ficarem fora de sincronia depois de um Create.
  onChange?: (areas: Area[]) => void;
};

// Áreas: a divisão de topo da vida do usuário à qual um Compromisso
// pertence (CONTEXT.md) — pré-requisito para criar qualquer Compromisso,
// por isso o formulário de Compromisso aponta para o cadastro quando não há nenhuma.
export function AreasSection({ onChange }: Props) {
  const [areas, setAreas] = useState<Area[]>([]);
  const [name, setName] = useState("");
  const [color, setColor] = useState("#6366f1");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [editingID, setEditingID] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editColor, setEditColor] = useState("#6366f1");
  const [busyID, setBusyID] = useState<string | null>(null);

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

  function startEdit(a: Area) {
    setError(null);
    setEditingID(a.id);
    setEditName(a.name);
    setEditColor(a.color || "#6366f1");
  }

  async function saveEdit(id: string) {
    setError(null);
    setBusyID(id);
    try {
      await api.put(`/areas/${id}`, { name: editName, color: editColor });
      setEditingID(null);
      await reload();
    } catch {
      setError("Não foi possível salvar — talvez o nome já exista.");
    } finally {
      setBusyID(null);
    }
  }

  async function remove(id: string) {
    setError(null);
    setBusyID(id);
    try {
      await api.delete(`/areas/${id}`);
      await reload();
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 409
          ? "Esta Área tem Compromissos associados — arquive-os antes de remover."
          : "Não foi possível remover a Área.",
      );
    } finally {
      setBusyID(null);
    }
  }

  return (
    <div className="mb-6 space-y-3 rounded-lg border border-slate-800 bg-slate-900 p-4">
      {areas.length > 0 && (
        <ul className="space-y-2">
          {areas.map((a) =>
            editingID === a.id ? (
              <li key={a.id} className="flex items-end gap-2">
                <input
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  className={`${inputClass} flex-1`}
                />
                <input
                  type="color"
                  value={editColor}
                  onChange={(e) => setEditColor(e.target.value)}
                  className="h-[38px] w-14 rounded-md border border-slate-700 bg-slate-800"
                />
                <button type="button" onClick={() => saveEdit(a.id)} disabled={busyID === a.id} className={buttonClass}>
                  Salvar
                </button>
                <button type="button" onClick={() => setEditingID(null)} className={buttonClass}>
                  Cancelar
                </button>
              </li>
            ) : (
              <li
                key={a.id}
                className="flex items-center justify-between gap-2 rounded-full border border-slate-700 px-3 py-1 text-sm text-slate-300"
              >
                <span className="flex items-center gap-1.5">
                  <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: a.color || "#94a3b8" }} />
                  {a.name}
                </span>
                <span className="flex gap-1">
                  <button
                    type="button"
                    onClick={() => startEdit(a)}
                    className="text-xs text-slate-400 hover:text-slate-100"
                  >
                    Editar
                  </button>
                  <button
                    type="button"
                    onClick={() => remove(a.id)}
                    disabled={busyID === a.id}
                    className="text-xs text-slate-400 hover:text-red-400 disabled:opacity-50"
                  >
                    Remover
                  </button>
                </span>
              </li>
            ),
          )}
        </ul>
      )}

      <form onSubmit={handleCreate} className="flex items-end gap-2">
        <div className="flex-1">
          <label htmlFor="area-name" className="block text-sm font-medium text-slate-300">
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
          <label htmlFor="area-color" className="block text-sm font-medium text-slate-300">
            Cor
          </label>
          <input
            id="area-color"
            type="color"
            value={color}
            onChange={(e) => setColor(e.target.value)}
            className="h-[38px] w-14 rounded-md border border-slate-700 bg-slate-800"
          />
        </div>
        <button type="submit" disabled={submitting} className={buttonClass}>
          Adicionar
        </button>
      </form>

      {error && <p className="text-sm text-red-400">{error}</p>}
    </div>
  );
}
