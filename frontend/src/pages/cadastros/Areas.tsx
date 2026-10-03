import { AreasSection } from "../../components/AreasSection";
import { Layout } from "../../components/Layout";

export function Areas() {
  return (
    <Layout>
      <h1 className="mb-1 text-lg font-semibold text-slate-100">Áreas</h1>
      <p className="mb-4 text-sm text-slate-400">
        A divisão de topo da sua vida. Todo compromisso pertence a uma Área.
      </p>
      <AreasSection />
    </Layout>
  );
}
