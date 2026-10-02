import { useEffect, useRef, useState } from "react";
import { api } from "../lib/api";
import type { InstanceQRCode, InstanceStatus } from "../lib/types";

const POLL_INTERVAL_MS = 5000;

// "not_provisioned" é sintético (não vem da EvolutionAPI): significa que a
// instância desta instalação ainda não foi criada — normal antes do
// primeiro pareamento, QRCode() provisiona automaticamente.
const STATE_LABELS: Record<string, string> = {
  open: "conectado",
  connecting: "conectando — escaneie o QR code",
  close: "desconectado",
  not_provisioned: "nunca configurado — escaneie o QR code para conectar",
};

// Painel de status/pareamento da ÚNICA instância WhatsApp desta instalação
// (ADR-0001 — instância, não por usuário: não existe uma instância por
// usuário aqui, é a mesma para todo mundo que usar esta tela).
export function WhatsAppInstancePanel() {
  const [state, setState] = useState<string | null>(null);
  const [qrCode, setQrCode] = useState<string | null>(null);
  const [error, setError] = useState(false);
  const [requestingQr, setRequestingQr] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // A EvolutionAPI (pelo menos v2.1.1, issues #2385/#2437 do projeto upstream)
  // entra num loop de reconexão e nunca produz QR code se /instance/connect
  // for chamado de novo antes do handshake anterior terminar. Por isso o QR
  // só é pedido uma vez por "sessão de pareamento" — não a cada tick do
  // polling — e um novo pedido depois disso é sempre uma ação explícita do
  // usuário (botão), nunca automática.
  const qrRequestedRef = useRef(false);

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
    setRequestingQr(true);
    try {
      const result = await api.get<InstanceQRCode>("/whatsapp-instance/qrcode");
      setQrCode(result.qrcode || null);
      setError(false);
    } catch {
      setError(true);
    } finally {
      setRequestingQr(false);
    }
  }

  useEffect(() => {
    async function tick() {
      const current = await refreshStatus();
      if (current === "open") {
        setQrCode(null);
        qrRequestedRef.current = false;
        return;
      }
      if (current && !qrRequestedRef.current) {
        qrRequestedRef.current = true;
        await requestQRCode();
      }
    }

    tick();
    pollRef.current = setInterval(tick, POLL_INTERVAL_MS);

    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function handleRequestNewQr() {
    qrRequestedRef.current = true;
    requestQRCode();
  }

  return (
    <div className="max-w-sm space-y-3 rounded-lg border border-slate-800 bg-slate-900 p-4">
      <h2 className="text-sm font-semibold text-slate-100">Instância WhatsApp</h2>

      {error && <p className="text-sm text-red-400">EvolutionAPI inacessível.</p>}

      {!error && (
        <p className="text-sm text-slate-300">
          Estado:{" "}
          <span className="font-medium text-slate-100">
            {state ? (STATE_LABELS[state] ?? state) : "consultando..."}
          </span>
        </p>
      )}

      {state !== "open" && qrCode && (
        <div>
          <img src={qrCode} alt="QR code para pareamento do WhatsApp" className="w-48 rounded-md bg-white p-2" />
          <p className="mt-1 text-xs text-slate-400">Escaneie com o WhatsApp do número desta instância.</p>
        </div>
      )}

      {state !== "open" && (
        <button
          type="button"
          onClick={handleRequestNewQr}
          disabled={requestingQr}
          className="rounded-md border border-slate-700 px-3 py-1.5 text-sm font-medium text-slate-300 hover:bg-slate-800 disabled:opacity-50"
        >
          {requestingQr ? "Gerando..." : qrCode ? "Gerar novo QR code" : "Gerar QR code"}
        </button>
      )}
    </div>
  );
}
