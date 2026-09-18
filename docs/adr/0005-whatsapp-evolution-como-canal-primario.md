---
status: accepted
---

# WhatsApp via EvolutionAPI é o primeiro Canal, apesar do risco

O primeiro Canal implementado de ponta a ponta é o WhatsApp, através de uma instância
self-hosted da EvolutionAPI. E-mail por SMTP entra junto, como fallback. Telegram,
Discord e Slack ficam para depois.

A decisão é deliberadamente contra o caminho tecnicamente confortável, e isso precisa
ficar registrado para que ninguém "conserte" depois. A recomendação de engenharia era
começar por Telegram: bot oficial, gratuito, webhook de entrada e botões inline, zero
risco de banimento. Foi rejeitada por uma razão de produto que supera as técnicas: a tese
inteira do Self Remember é que a Cobrança chega onde o usuário **realmente lê**. Validar o
motor em um canal que o usuário abre três vezes por dia não valida coisa alguma — só adia
a descoberta de que a entrega no WhatsApp tem restrições que mudam o desenho.

## Riscos aceitos

- A EvolutionAPI automatiza um cliente WhatsApp **não oficial**. Contas que disparam
  mensagens automatizadas podem ser banidas, inclusive em testes.
- A sessão é pareada por QR code e cai sozinha. Ela pode morrer às 3h da manhã e o produto
  para de cobrar em silêncio.
- É mais um serviço com estado para operar no compose.

## Mitigações obrigatórias, não opcionais

- **Health check da instância Evolution a cada minuto.** Sessão desconectada é um
  incidente que dispara alerta pelo canal de e-mail, que não depende dela.
- **Cadeia de fallback obrigatória**: Entrega que falha ou fica sem confirmação do
  provedor escala para o próximo Canal verificado do usuário. Nenhum Compromisso pode ter
  o WhatsApp como único Canal.
- **Teto de mensagens por minuto por instância**, com espaçamento aleatório entre envios.
- A porta `Channel` é desenhada de forma que Telegram seja um adaptador de ~150 linhas.
  Quando (não se) a Evolution der problema, a troca custa horas, não semanas.
