import { useEffect, useState, type FormEvent } from "react";
import { api } from "../lib/api";
import type { Channel, ChannelKind } from "../lib/types";

const inputClass =
  "w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500";
const buttonClass =
  "rounded-md border border-slate-700 px-3 py-1 text-sm text-slate-300 hover:bg-slate-800 disabled:opacity-50";

// Canais: os contatos e e-mails alvo do aviso (ARCHITECTURE.md §6) — não a
// instância WhatsApp em si (ver WhatsAppInstancePanel), e sim os endereços
// que recebem a Cobrança depois que ela sai da instância/SMTP.
export function ChannelsSection() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [kind, setKind] = useState<ChannelKind>("email");
  const [address, setAddress] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [verifyingID, setVerifyingID] = useState<string | null>(null);
  const [code, setCode] = useState("");
  const [busyID, setBusyID] = useState<string | null>(null);

  async function reload() {
    setChannels(await api.get<Channel[]>("/channels"));
  }

  useEffect(() => {
    reload().catch(console.error);
  }, []);

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await api.post("/channels", { kind, address });
      setAddress("");
      await reload();
    } catch {
      setError("Não foi possível adicionar o Canal — confira o endereço.");
    }
  }

  async function remove(id: string) {
    setBusyID(id);
    try {
      await api.delete(`/channels/${id}`);
      await reload();
    } finally {
      setBusyID(null);
    }
  }

  async function requestVerification(id: string) {
    setError(null);
    setBusyID(id);
    try {
      await api.post(`/channels/${id}/verify/request`);
      setVerifyingID(id);
    } catch {
      setError("Não foi possível enviar o código — o Canal (SMTP/WhatsApp) pode estar indisponível.");
    } finally {
      setBusyID(null);
    }
  }

  async function confirmVerification(id: string) {
    if (!code) return;
    setError(null);
    setBusyID(id);
    try {
      await api.post(`/channels/${id}/verify/confirm`, { code });
      setVerifyingID(null);
      setCode("");
      await reload();
    } catch {
      setError("Código incorreto ou expirado — solicite um novo.");
    } finally {
      setBusyID(null);
    }
  }

  return (
    <div className="max-w-sm space-y-4 rounded-lg border border-slate-800 bg-slate-900 p-4">
      <ul className="space-y-2">
        {channels.map((ch) => (
          <li key={ch.id} className="rounded-md border border-slate-800 p-3">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-slate-100">{ch.address}</p>
                <p className="text-xs text-slate-400">
                  {ch.kind === "email" ? "E-mail" : "WhatsApp"} ·{" "}
                  {ch.verified ? "verificado" : "não verificado"}
                </p>
              </div>
              <div className="flex gap-2">
                {!ch.verified && verifyingID !== ch.id && (
                  <button
                    type="button"
                    onClick={() => requestVerification(ch.id)}
                    disabled={busyID === ch.id}
                    className={buttonClass}
                  >
                    Verificar
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => remove(ch.id)}
                  disabled={busyID === ch.id}
                  className={buttonClass}
                >
                  Remover
                </button>
              </div>
            </div>

            {verifyingID === ch.id && (
              <div className="mt-2 flex gap-2">
                <label htmlFor={`code-${ch.id}`} className="sr-only">
                  Código de verificação
                </label>
                <input
                  id={`code-${ch.id}`}
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  placeholder="Código recebido"
                  className={inputClass}
                />
                <button
                  type="button"
                  onClick={() => confirmVerification(ch.id)}
                  disabled={busyID === ch.id}
                  className={buttonClass}
                >
                  Confirmar
                </button>
              </div>
            )}
          </li>
        ))}
      </ul>

      <form onSubmit={handleCreate} className="space-y-2 border-t border-slate-800 pt-3">
        <div className="flex gap-2">
          <label htmlFor="channel-kind" className="sr-only">
            Tipo de Canal
          </label>
          <select
            id="channel-kind"
            value={kind}
            onChange={(e) => setKind(e.target.value as ChannelKind)}
            className={inputClass}
          >
            <option value="email">E-mail</option>
            <option value="whatsapp">WhatsApp</option>
          </select>
          <label htmlFor="channel-address" className="sr-only">
            Endereço
          </label>
          <input
            id="channel-address"
            required
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            placeholder={kind === "email" ? "voce@exemplo.com" : "+55 11 91234-5678"}
            className={inputClass}
          />
        </div>
        <button type="submit" className={buttonClass}>
          Adicionar Canal
        </button>
      </form>

      {error && <p className="text-sm text-red-400">{error}</p>}
    </div>
  );
}
