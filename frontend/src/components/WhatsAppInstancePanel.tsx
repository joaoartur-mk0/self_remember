import { useEffect, useRef, useState } from "react";
import { api } from "../lib/api";
import type { InstanceQRCode, InstanceStatus } from "../lib/types";

const POLL_INTERVAL_MS = 5000;

// Painel de status/pareamento da ÚNICA instância WhatsApp desta instalação
// (ADR-0001 — instância, não por usuário: não existe uma instância por
// usuário aqui, é a mesma para todo mundo que usar esta tela).
export function WhatsAppInstancePanel() {
  const [state, setState] = useState<string | null>(null);
  const [qrCode, setQrCode] = useState<string | null>(null);
  const [error, setError] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  async function refreshStatus() {
    try {
      const status = await api.get<InstanceStatus>("/whatsapp-instance/status");
      setState(status.state);
      setError(false);
      return status.state;
    } catch {
      setError(true);
      return null;
    }
  }

  async function requestQRCode() {
    try {
      const result = await api.get<InstanceQRCode>("/whatsapp-instance/qrcode");
      setQrCode(result.qrcode || null);
      setError(false);
    } catch {
      setError(true);
    }
  }

  useEffect(() => {
    refreshStatus();

    pollRef.current = setInterval(async () => {
      const current = await refreshStatus();
      if (current !== "open") {
        await requestQRCode();
      } else {
        setQrCode(null);
      }
    }, POLL_INTERVAL_MS);

    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="max-w-sm space-y-3 rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="text-sm font-semibold text-slate-900">Instância WhatsApp</h2>

      {error && <p className="text-sm text-red-600">EvolutionAPI inacessível.</p>}

      {!error && (
        <p className="text-sm text-slate-700">
          Estado: <span className="font-medium">{state ?? "consultando..."}</span>
        </p>
      )}

      {state !== "open" && qrCode && (
        <div>
          <img src={qrCode} alt="QR code para pareamento do WhatsApp" className="w-48" />
          <p className="mt-1 text-xs text-slate-500">Escaneie com o WhatsApp do número desta instância.</p>
        </div>
      )}
    </div>
  );
}
