# Ocorrências são materializadas no banco, não calculadas sob demanda

Cada Ocorrência de um Compromisso recorrente é gravada como uma linha real na tabela
`occurrences`, criada antecipadamente por um job periódico (o Materializador) dentro de um
horizonte de 48 horas. Não calculamos "quais lembretes vencem agora" expandindo regras de
recorrência em tempo de consulta.

A alternativa — guardar só a regra e expandir sob demanda — é mais enxuta e tem um defeito
fatal para este produto: a Ocorrência é a entidade que carrega estado. Ela é cobrada,
escalada, adiada, reconhecida ou abandonada. Tudo isso é histórico por instância, e
histórico por instância precisa de uma linha. Sem ela, não conseguimos responder "por que
você não foi cobrado no dia 14?", que é a pergunta de suporte número um de um sistema de
lembretes, nem manter a Política de Acúmulo, que depende de saber que a ocorrência de
ontem ainda está viva.

## Consequências

- Editar um Compromisso recorrente exige decidir o que fazer com as Ocorrências futuras já
  materializadas. A regra: Ocorrências futuras **ainda não vencidas e sem Entregas** são
  regeradas; as demais são preservadas como estão. O passado nunca é reescrito.
- O horizonte de materialização (48h) é uma constante operacional: aumentá-lo aumenta o
  custo de regerar após edições.
- A tabela cresce indefinidamente e precisa de política de retenção/particionamento antes
  do primeiro ano.
