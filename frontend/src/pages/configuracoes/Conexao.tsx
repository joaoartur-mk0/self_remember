import { Layout } from "../../components/Layout";
import { WhatsAppInstancePanel } from "../../components/WhatsAppInstancePanel";

export function Conexao() {
  return (
    <Layout>
      <h1 className="mb-1 text-lg font-semibold text-slate-100">Conexão WhatsApp</h1>
      <p className="mb-4 text-sm text-slate-400">Estado e pareamento da instância da EvolutionAPI.</p>
      <WhatsAppInstancePanel />
    </Layout>
  );
}
