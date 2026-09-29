# Self Remember

Para quem tem mais tarefas do que a mente consegue lembrar.

O Self Remember não é mais um quadro de cards bonitos que você abandona em duas semanas.
A premissa não é **organizar**, é **cobrar**: seus compromissos chegam até você nos canais
que você realmente lê — WhatsApp, e-mail e, no futuro, Telegram, Discord e Slack — com
insistência proporcional à importância, até que você diga o que fez com eles.

E "dizer o que fez" tem três respostas igualmente legítimas: **concluí**, **adio para
tal hora** ou **não vou fazer**. A terceira é tão de primeira classe quanto as outras, de
propósito. Um sistema que só aceita "concluí" ensina o usuário a mentir, e um sistema que
recebe mentiras não sabe mais a diferença entre o que você faz e o que você só diz que vai
fazer.

> **Estado: documentação. Nada foi implementado ainda.**

## Por que ele existe

O maior obstáculo para se organizar não é a falta do Kanban perfeito. É a fuga das
distrações e a incapacidade de manter na cabeça mais tarefas do que a memória comporta.
O Self Remember ataca isso pela entrega, não pela visualização.

Corolário que guia todas as decisões do projeto: o inimigo não é o esquecimento, é a
**fadiga de alerta**. Um sistema que grita o tempo todo tem a mesma relação sinal/ruído de
um que nunca grita, e falha de um jeito pior — você silencia o canal e não há erro nenhum
para avisar que o produto morreu. Por isso a insistência é escolhida por compromisso, e
por isso existe uma janela de silêncio que nada atravessa.

## Documentação

| Arquivo | Conteúdo |
|---|---|
| [`CONTEXT.md`](./CONTEXT.md) | Glossário do domínio. **Normativo**: os nomes daqui são os nomes do código. |
| [`ARCHITECTURE.md`](./ARCHITECTURE.md) | Estrutura do sistema, motor de cobrança, modelo de dados, deploy. |
| [`docs/adr/`](./docs/adr/) | Decisões arquiteturais e por que foram tomadas. |
| [`SELF_REMEMBER.md`](./SELF_REMEMBER.md) | Documento original de conceito, preservado como registro da ideia. |
| [`CHANGELOG.md`](./CHANGELOG.md) | Histórico de mudanças. |

Antes de escrever código, leia `CONTEXT.md`. Ele é curto e evita o problema mais caro de
um projeto assim: três nomes diferentes para a mesma coisa em três camadas.

## Stack

**Backend** — Go, Gin, GORM, PostgreSQL, [River](https://riverqueue.com) (fila sobre o
próprio Postgres, sem broker separado), migrations com goose, Swagger.

**Frontend** — Vite, React, Tailwind.

**Canais** — [EvolutionAPI](https://github.com/EvolutionAPI/evolution-api) (WhatsApp) e
SMTP no MVP. Telegram, Discord e Slack depois, atrás da mesma porta `Channel`.

**Deploy** — Docker Compose, local e produção.

## Como funciona, em um parágrafo

Você cria um **Compromisso** (com ou sem recorrência), diz a que **Área** da sua vida ele
pertence e quão duro quer ser cobrado. Um job periódico materializa as **Ocorrências**
dos próximos dois dias. A cada minuto, o **Scanner** procura ocorrências vencidas sem
reconhecimento e enfileira **Entregas** pelos seus canais, subindo os degraus da escada de
insistência. Você abre o app e **reconhece** — concluindo, adiando ou desistindo. Se nunca
reconhecer, a ocorrência é **abandonada** depois de esgotar a escada, e o sistema passa a
sugerir que aquele compromisso talvez não devesse existir.

## Rodando localmente

```bash
cp .env.example .env          # preencha os segredos
docker compose -f docker-compose.local.yml up -d --build
```

- Frontend: http://localhost:5173
- API: http://localhost:8080 — Swagger em `/swagger/index.html`
- Postgres: `localhost:5432`

Não existe usuário padrão nem seed — o cadastro é fechado por padrão (ADR-0001). O
primeiro usuário é criado pelo `cmd/admin`, que não sobe com `up` (perfil `tools`,
`docker compose run`, nunca um serviço de longa duração):

```bash
docker compose --profile tools -f docker-compose.local.yml run --rm admin \
  create-user --email voce@exemplo.com --name "Seu Nome" --timezone America/Sao_Paulo
# prompt "senha:" — digite e Enter
```

Cada Canal (WhatsApp/e-mail) precisa ser verificado em Ajustes antes de receber a
primeira cobrança de verdade.

## Ordem sugerida de construção

Cada etapa termina em algo demonstrável. A ordem não é negociável nos dois primeiros
itens: o motor precisa existir antes de qualquer tela, porque é ele o risco do projeto.

1. **Fundação** — módulo, migrations, `users`, autenticação, health check, Swagger.
2. **Domínio puro** — compromissos, recorrência com fuso, máquina de estados da ocorrência,
   escada de insistência, janela de silêncio. Tudo com relógio injetado e coberto por
   testes. Nenhuma linha de rede aqui.
3. **Motor** — River, Materializador, Scanner, Guardião, e um canal falso que só grava em
   log. Neste ponto o produto já funciona de verdade; só não sai do servidor.
4. **Canais reais** — SMTP primeiro (é o fallback e não pode falhar), depois EvolutionAPI
   com health check e cadeia de fallback.
5. **API e frontend** — CRUD, e a tela *Hoje* com os três botões de reconhecimento.
6. **Produção** — compose de prod com Traefik, logs, backup do Postgres e do volume de
   sessão da Evolution.

## Decisões que provavelmente vão te surpreender

Estão todas registradas em [`docs/adr/`](./docs/adr/) com o motivo. As três que mais
costumam gerar "por que não fizeram do jeito óbvio?":

- **Nenhum handler HTTP enfileira job.** Só o Scanner produz trabalho, e o banco é a única
  fonte de verdade sobre o que deve ser cobrado. ([ADR-0003](./docs/adr/0003-scanner-como-unico-produtor-de-jobs.md))
- **`gorm.AutoMigrate` é proibido**, inclusive em ambiente local. ([ADR-0007](./docs/adr/0007-migrations-versionadas-em-vez-de-gorm-automigrate.md))
- **WhatsApp vem antes do Telegram**, contra a recomendação técnica, por uma razão de
  produto — com riscos e mitigações obrigatórias escritas. ([ADR-0005](./docs/adr/0005-whatsapp-evolution-como-canal-primario.md))
