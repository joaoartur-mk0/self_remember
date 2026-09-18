# Self Remember

Sistema de compromissos pessoais cuja função não é organizar, e sim **cobrar**: garantir
que o usuário seja lembrado, com insistência proporcional à importância, até que ele
reconheça o compromisso — concluindo, adiando ou desistindo dele explicitamente.

Este arquivo é **somente glossário**. Decisões técnicas ficam em `docs/adr/`, e a
estrutura do sistema em `ARCHITECTURE.md`.

## Convenção de idioma

Os termos canônicos do domínio são definidos em português, porque é assim que o produto
é pensado e falado. No código Go, cada termo tem **um** identificador em inglês fixado
aqui entre parênteses. Nunca invente um sinônimo em código: se um conceito novo aparecer,
ele entra neste arquivo antes de entrar no código.

## Language

### Compromisso e classificação

**Compromisso** (`Commitment`):
Algo que o usuário se comprometeu a fazer, com ou sem recorrência. É o registro
duradouro da intenção — não é a coisa que dispara no relógio.
_Avoid_: tarefa, task, todo, card, item, lead

**Ocorrência** (`Occurrence`):
Uma instância datada de um Compromisso. É a unidade que vence, que é cobrada e que é
reconhecida. Um Compromisso único tem uma Ocorrência; um recorrente tem muitas.
_Avoid_: instância, agendamento, evento, disparo

**Área** (`Area`):
A divisão de topo da vida do usuário à qual um Compromisso pertence — Trabalho, Pessoal,
Saúde. Toda Ocorrência herda a Área do seu Compromisso, e é por Área que o usuário separa
o que pode ser cobrado em qual horário.
_Avoid_: categoria, projeto, contexto, pipeline

**Etiqueta** (`Label`):
Classificação livre e transversal, aplicável a Compromissos de qualquer Área. Ao
contrário da Área, não influencia a cobrança.
_Avoid_: tag, marcador, categoria

### Cobrança

**Cobrança** (`Nag`):
A tentativa lógica de fazer o usuário agir sobre uma Ocorrência vencida. Escolhemos
"cobrança" e não "aviso" de propósito: um aviso é informativo e pode ser ignorado sem
consequência, enquanto a cobrança é o mecanismo que insiste e escala. Se o código disser
"notification", o próximo desenvolvedor vai tratá-la como descartável.
_Avoid_: aviso, notificação, alerta, lembrete, ping

**Entrega** (`Delivery`):
Uma tentativa física de fazer uma Cobrança chegar ao usuário por **um** Canal
específico. Uma única Cobrança pode gerar várias Entregas (WhatsApp falhou, tentou
e-mail). É o registro que prova o que foi enviado, quando e com que resultado.
_Avoid_: envio, mensagem, tentativa

**Canal** (`Channel`):
Um meio de entrega configurado e verificado para um usuário — um número de WhatsApp, um
endereço de e-mail, um chat do Telegram. Um Canal só pode receber Entregas depois de
Verificado.
_Avoid_: integração, provedor, destino

**Perfil de Insistência** (`NagProfile`):
A política que define quão duro o sistema cobra uma Ocorrência: quantas vezes, com que
intervalos e por quais Canais. É escolhido por Compromisso, não globalmente. Os três
perfis do sistema são Leve, Normal e Implacável.
_Avoid_: prioridade, urgência, severidade, política

**Degrau** (`NagStep`):
Um nível da escada de um Perfil de Insistência: um atraso desde o vencimento e o Canal a
usar naquele nível. Degraus mais altos são mais intrusivos.
_Avoid_: nível, etapa, retry, tentativa

**Janela de Silêncio** (`QuietHours`):
O intervalo diário, definido pelo usuário no seu fuso, em que nenhuma Entrega é feita.
Cobranças que venceriam dentro dela não são perdidas: são represadas e entregues na
primeira brecha. Nem o perfil Implacável a atravessa.
_Avoid_: modo não perturbe, DND, silêncio, pausa

**Escalonamento** (`Escalation`):
A subida de um Degrau para o próximo quando uma Ocorrência continua sem Reconhecimento.
_Avoid_: retry, reenvio, insistência

### Ciclo de vida

**Reconhecimento** (`Ack`):
O ato do usuário que encerra a cobrança de uma Ocorrência. Existem exatamente três
formas: Conclusão, Adiamento e Desistência. Não reconhecer não é uma quarta forma — é a
ausência de todas elas, e é o que mantém a escada subindo.
_Avoid_: resposta, confirmação, interação

**Conclusão** (`Completion`):
O Reconhecimento que afirma que a Ocorrência foi cumprida.
_Avoid_: check, done, finalização, marcar

**Adiamento** (`Snooze`):
O Reconhecimento que empurra a Ocorrência para um novo vencimento, preservando o
histórico de Entregas já feitas. Adiar não zera a escada — uma Ocorrência adiada três
vezes retoma a cobrança de onde parou, não do primeiro Degrau.
_Avoid_: postergar, remarcar, reagendar

**Desistência** (`Dismissal`):
O Reconhecimento em que o usuário declara que **não vai** cumprir a Ocorrência. É de
primeira classe e nunca deve ser desencorajada na interface: sem uma saída honesta, o
usuário conclui mentindo, e o sistema perde a capacidade de distinguir o que ele faz do
que ele só diz que vai fazer.
_Avoid_: cancelar, recusar, ignorar, deletar

**Abandono** (`Abandonment`):
O estado terminal de uma Ocorrência que esgotou todos os Degraus do seu Perfil sem
nenhum Reconhecimento. É diferente de Desistência: aqui o usuário nunca se manifestou.
Abandono repetido no mesmo Compromisso é o sinal de que ele deveria ser arquivado.
_Avoid_: expirado, perdido, falho, vencido

**Política de Acúmulo** (`CarryOverPolicy`):
Define o que acontece com uma Ocorrência não reconhecida quando a próxima Ocorrência do
mesmo Compromisso recorrente vence. `stack` mantém as duas cobrando (tomar o remédio de
ontem ainda importa); `replace` encerra a antiga como Abandonada (regar a planta duas
vezes não faz sentido). É uma propriedade do Compromisso.
_Avoid_: acumular, empilhar, backlog

**Represamento** (`Backlog`):
O conjunto de Ocorrências vencidas que não foram entregues porque a Janela de Silêncio
estava ativa ou o usuário esteve inalcançável. Represamento tem prazo: passado o
Horizonte de Relevância, a Ocorrência é Abandonada em vez de entregue, para que voltar de
uma viagem não produza uma avalanche.
_Avoid_: fila, pendências, atrasados
