---
status: accepted
---

# Reconhecimento só pelo app web no MVP, com o modelo preparado para ACK por Canal

No MVP, as três formas de Reconhecimento — Conclusão, Adiamento e Desistência —
acontecem exclusivamente no app web. A Cobrança entregue no WhatsApp ou no e-mail carrega
um deep link para a Ocorrência; responder a mensagem não faz nada.

A objeção levantada e conscientemente aceita: isso exige uma troca de contexto de dezenas
de segundos para encerrar uma Cobrança de dois segundos, e a pessoa que este produto
atende — a que tem mais tarefas do que memória — é exatamente a que vai ler a mensagem,
pensar "já fiz isso" e não abrir o app. O sistema então cobra de novo algo já feito, que é
o pior estado possível: insistente e errado. Em troca, o MVP não precisa de ingestão de
webhooks de entrada, correlação resposta→Ocorrência nem tokens de ação assinados — um
subsistema inteiro a menos antes de o motor estar de pé.

## O que fazemos agora para que isso não seja uma dívida cara

O modelo de dados já assume ACK por Canal, mesmo sem implementá-lo:

- `acks` é uma tabela própria com `source` (`web` | `channel` | `system`) e
  `channel_id` nulo. Nada muda no schema quando o ACK por Canal chegar.
- A porta `Channel` já declara `SupportsInboundAck() bool`; todos os adaptadores do MVP
  retornam `false`.
- Toda Entrega grava um `correlation_id` no metadado da mensagem enviada. Sem ele, nunca
  seria possível correlacionar uma resposta futura à Ocorrência que a originou — e esse é
  o dado que não dá para reconstruir retroativamente.

## Sinal para reverter

Se a taxa de Abandono passar de ~30% das Ocorrências entregues, a causa mais provável não
é o usuário não fazer as coisas: é ele fazer e não reconhecer. Esse número é o gatilho
para implementar o ACK por Canal, e por isso ele é métrica de primeira classe desde o
primeiro dia.
