# O Scanner é o único produtor de jobs de Entrega

Nenhum handler HTTP enfileira job de cobrança. Criar, editar ou adiar um Compromisso
apenas escreve no Postgres. Um job periódico do River (o Scanner), rodando a cada minuto,
é o único componente que lê Ocorrências vencidas e enfileira Entregas.

Essa escolha existe para resolver um problema concreto de acoplamento. A API usa GORM
(sobre `database/sql`) e o River exige um pool `pgx/v5` nativo. Enfileirar um job
atomicamente junto de uma escrita da API exigiria `river.InsertTx` com uma transação pgx —
ou seja, a camada de escrita da API teria que abandonar o GORM ou manter duas transações
com um commit em duas fases improvisado. Ambos os caminhos são ruins, e a falha deles é
silenciosa: um Compromisso criado sem job é um Compromisso que nunca cobra, exatamente o
fracasso que o produto não pode ter.

Ao fazer do estado do banco a única fonte de verdade sobre o que deve ser cobrado, o
enfileiramento passa a ser derivado e reconstruível. Se a fila for perdida inteira, o
próximo tick do Scanner reconstitui tudo.

## Consequências

- Latência de até 60 segundos entre o vencimento e a primeira Entrega. Aceitável: nenhuma
  Cobrança deste sistema é sensível ao segundo.
- O Scanner é um ponto único de falha do produto inteiro. Precisa de alerta próprio: se
  ele não rodar por 5 minutos, isso é um incidente.
- `river.InsertTx` ainda é usado **dentro** do Scanner, que roda em pgx puro e enfileira as
  Entregas na mesma transação em que marca as Ocorrências como enfileiradas.
