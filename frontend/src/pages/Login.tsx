import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { api } from "../lib/api";

export function Login() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await api.post("/auth/login", { email, password });
      navigate("/", { replace: true });
    } catch {
      setError("E-mail ou senha inválidos.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-50">
      <form onSubmit={handleSubmit} className="w-full max-w-sm space-y-4 rounded-lg border border-slate-200 bg-white p-6">
        <h1 className="text-lg font-semibold text-slate-900">Entrar</h1>

        <div className="space-y-1">
          <label htmlFor="email" className="block text-sm font-medium text-slate-700">
            E-mail
          </label>
          <input
            id="email"
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
          />
        </div>

        <div className="space-y-1">
          <label htmlFor="password" className="block text-sm font-medium text-slate-700">
            Senha
          </label>
          <input
            id="password"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm"
          />
        </div>

        {error && <p className="text-sm text-red-600">{error}</p>}

        <button
          type="submit"
          disabled={submitting}
          className="w-full rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          {submitting ? "Entrando..." : "Entrar"}
        </button>
      </form>
    </div>
  );
}
