import { ChannelsSection } from "../../components/ChannelsSection";
import { Layout } from "../../components/Layout";

export function Canais() {
  return (
    <Layout>
      <h1 className="mb-1 text-lg font-semibold text-slate-100">Canais</h1>
      <p className="mb-4 text-sm text-slate-400">E-mails e números que recebem as cobranças.</p>
      <ChannelsSection />
    </Layout>
  );
}
