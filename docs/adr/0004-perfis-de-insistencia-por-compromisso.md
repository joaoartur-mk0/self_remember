# Insistência é um Perfil escolhido por Compromisso, com Janela de Silêncio global

Cada Compromisso escolhe um Perfil de Insistência — Leve, Normal ou Implacável — que
define a escada de Degraus usada para cobrá-lo. A Janela de Silêncio é configurada uma vez
por usuário, no fuso dele, e nenhum perfil a atravessa.

| Perfil | Degraus | Teto | Fim sem Reconhecimento |
|---|---|---|---|
| Leve | 1 Entrega no vencimento | 1 | Abandonada |
| Normal | +0, +15min, +1h, +4h, +1d | 5 | Abandonada |
| Implacável | +0, +5min, +15min, +30min, depois a cada 1h | sem teto | segue até Reconhecimento ou Horizonte de Relevância |

A alternativa descrita no conceito original era insistência infinita para tudo. Rejeitada:
um sistema que grita sempre tem a mesma relação sinal/ruído de um que nunca grita, e o modo
de falha é irreversível — o usuário silencia o canal e o produto morre sem emitir erro
nenhum. Colocar a escolha no Compromisso preserva a promessa ("existe um modo que não
desiste") sem gastar o único recurso escasso do produto, que é a atenção do usuário.

O perfil Implacável não tem teto de tentativas, mas tem duas contenções que o impedem de
virar um loop infinito de verdade: a Janela de Silêncio e o Horizonte de Relevância (uma
Ocorrência vencida há mais de 7 dias é Abandonada em vez de entregue).

## Consequências

- Os Perfis são constantes do sistema, não editáveis pelo usuário no MVP. Perfis
  customizados podem vir depois sem quebrar o modelo, porque o Degrau já é uma entidade.
- A escada é retomada, não reiniciada, após um Adiamento — ver `CONTEXT.md`.
- Cobranças represadas pela Janela de Silêncio saem enfileiradas na primeira brecha, com
  agrupamento por Área para não produzir uma rajada de mensagens às 8h01.
