import { useEffect, useState, type ReactNode } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { api, ApiError } from "./lib/api";
import { Areas } from "./pages/cadastros/Areas";
import { Compromissos } from "./pages/Compromissos";
import { Canais } from "./pages/configuracoes/Canais";
import { Conexao } from "./pages/configuracoes/Conexao";
import { Preferencias } from "./pages/configuracoes/Preferencias";
import { Hoje } from "./pages/Hoje";
import { Login } from "./pages/Login";
import { NovoCompromisso } from "./pages/NovoCompromisso";
import { Painel } from "./pages/Painel";

// RequireAuth confirma sessão com uma chamada autenticada leve antes de
// renderizar a rota — sem sessão (401 mesmo após tentativa de refresh em
// src/lib/api.ts), redireciona para /login. Reaproveita GET /occurrences/today
// em vez de um GET /me dedicado, por ser mais barato de adicionar agora.
function RequireAuth({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<"checking" | "ok" | "denied">("checking");

  useEffect(() => {
    let active = true;
    api
      .get("/occurrences/today")
      .then(() => active && setStatus("ok"))
      .catch((err) => {
        if (!active) return;
        // 401 sem sessão renderiza <Navigate>; qualquer outro erro (ex.: 404
        // por Ajustes ainda não configurados) não impede o acesso à rota.
        setStatus(err instanceof ApiError && err.status === 401 ? "denied" : "ok");
      });
    return () => {
      active = false;
    };
  }, []);

  if (status === "checking") return null;
  if (status === "denied") return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function protectedRoute(path: string, element: ReactNode) {
  return <Route path={path} element={<RequireAuth>{element}</RequireAuth>} />;
}

export function AppRoutes() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        {protectedRoute("/", <Painel />)}
        {protectedRoute("/hoje", <Hoje />)}
        {protectedRoute("/compromissos", <Compromissos />)}
        {protectedRoute("/compromissos/novo", <NovoCompromisso />)}
        {protectedRoute("/cadastros/areas", <Areas />)}
        {protectedRoute("/configuracoes/preferencias", <Preferencias />)}
        {protectedRoute("/configuracoes/canais", <Canais />)}
        {protectedRoute("/configuracoes/conexao", <Conexao />)}
        {/* Ajustes foi dividido em Preferências, Canais e Conexão. */}
        <Route path="/ajustes" element={<Navigate to="/configuracoes/preferencias" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
