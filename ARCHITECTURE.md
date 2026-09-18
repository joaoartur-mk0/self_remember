# Arquitetura — Self Remember

Este documento descreve a estrutura do sistema. O vocabulário do domínio está em
[`CONTEXT.md`](./CONTEXT.md) e é normativo: os nomes usados aqui e no código vêm de lá.
As decisões que explicam *por que* a arquitetura é assim estão em [`docs/adr/`](./docs/adr/).

## 1. O que o sistema faz

O Self Remember transforma Compromissos em Ocorrências datadas, detecta quais venceram e
as cobra por Canais externos com insistência crescente, até que o usuário reconheça a
Ocorrência ou ela seja Abandonada.

Todo o resto — CRUD, autenticação, painel, classificação por Área — existe para alimentar
esse ciclo. Se algo na base de código não serve ao ciclo de cobrança, é acessório.

## 2. Visão geral

```
                 ┌──────────────┐
   navegador ───▶│   Frontend   │  Vite + React + Tailwind
                 │  (nginx)     │
                 └──────┬───────┘
                        │ REST /api/v1 (JWT)
                 ┌──────▼───────┐
                 │     API      │  Go + Gin + GORM      cmd/api
                 │   (stateless)│  escreve estado, nunca enfileira
                 └──────┬───────┘
                        │
                 ┌──────▼───────────────────────────────┐
                 │            PostgreSQL                │
                 │  domínio (GORM)  +  river_job (pgx)  │
                 └──────▲───────────────────────────────┘
                        │
                 ┌──────┴───────┐
                 │    Worker    │  Go + River            cmd/worker
                 │              │  Materializador, Scanner,
                 │              │  Entregador, Guardião
                 └──────┬───────┘
                        │
        ┌───────────────┼────────────────┐
        ▼               ▼                ▼
  EvolutionAPI       SMTP          (Telegram, Discord,
   (WhatsApp)       (e-mail)        Slack — futuro)
```

**API e Worker são processos separados**, do mesmo binário-base e do mesmo módulo Go. A
separação não é cerimônia: uma chamada lenta à EvolutionAPI não pode consumir o pool de
requisições HTTP, e o Worker precisa escalar e reiniciar por conta própria.

## 3. O motor de cobrança

É o coração do sistema e consiste em quatro jobs do River, três deles periódicos.

### 3.1 Materializador — `PeriodicInterval(1h)`

Lê Compromissos ativos e cria as Ocorrências que vencem nas próximas **48 horas**
(Horizonte de Materialização), se ainda não existirem. É idempotente por um índice único
em `(commitment_id, due_at)`.

A expansão da recorrência é feita **no fuso do usuário**, não em UTC. Uma regra "todo dia
às 8h" significa 8h no relógio de parede: na virada do horário de verão, somar 24 horas ao
UTC anterior produziria 7h ou 9h. O `due_at` é persistido em `timestamptz` (UTC) junto com
o `timezone` IANA que o gerou, para que o cálculo seja auditável.

### 3.2 Scanner — `PeriodicInterval(1m)`

O único produtor de jobs de Entrega do sistema ([ADR-0003](./docs/adr/0003-scanner-como-unico-produtor-de-jobs.md)).
A cada minuto, em uma transação pgx:

1. Seleciona Ocorrências elegíveis: vencidas, sem Reconhecimento, não Abandonadas, cujo
   próximo Degrau já está maduro, fora da Janela de Silêncio do usuário e dentro do
   Horizonte de Relevância (7 dias).
2. Para cada uma, enfileira uma Entrega com `river.InsertTx` e avança
   `occurrences.current_step`.
3. Commita. Se o commit falhar, nada foi enfileirado e nada avançou.

O `SELECT` usa `FOR UPDATE SKIP LOCKED`, de modo que múltiplas réplicas do Worker podem
rodar sem cobrar em dobro.

### 3.3 Entregador — job sob demanda

Recebe `(occurrence_id, step_index, channel_id)` e faz uma tentativa física de entrega.

A idempotência é garantida **no banco, antes da chamada externa**, e não pela fila. O
Entregador insere a linha em `deliveries` com `UNIQUE (occurrence_id, step_index,
channel_id)` primeiro; se a inserção colidir, a Entrega já foi feita e o job termina com
sucesso sem enviar nada. Só depois disso a API externa é chamada, e o resultado atualiza a
linha. Sem isso, uma retentativa do River após um envio bem-sucedido cujo ACK se perdeu
entregaria a mesma Cobrança duas vezes — indistinguível, para o usuário, de um bug de
loop, em um produto cuja tese inteira é a insistência controlada.

Em falha definitiva do Canal, o Entregador não estoura: ele registra a falha e enfileira
uma Entrega no próximo Canal verificado do usuário (cadeia de fallback,
[ADR-0005](./docs/adr/0005-whatsapp-evolution-como-canal-primario.md)).

### 3.4 Guardião — `PeriodicInterval(5m)`

Aplica os estados terminais que ninguém mais aplica:

- Ocorrências que esgotaram a escada do seu Perfil → **Abandonadas**.
- Ocorrências vencidas além do Horizonte de Relevância → **Abandonadas**, nunca entregues.
  É isso que impede que voltar de uma viagem de uma semana produza uma avalanche.
- Compromissos recorrentes com `carry_over = replace` → a Ocorrência anterior é Abandonada
  quando a nova vence.
- Compromissos com Abandono repetido → marcados para sugestão de arquivamento no painel.
- Health check da instância EvolutionAPI; sessão caída dispara alerta por e-mail.

## 4. Modelo de dados

Apenas a forma; o DDL fica em `backend/migrations/`.

| Tabela | Papel | Pontos não óbvios |
|---|---|---|
| `users` | conta | senha com argon2id |
| `user_settings` | preferências | `timezone` (IANA, obrigatório), `quiet_hours_start/end`, Canal padrão |
| `channels` | Canais do usuário | `kind`, `address`, `config` cifrado, `verified_at`, `fallback_order` |
| `areas` | Áreas de vida | `name`, `color`, Janela de Silêncio própria opcional |
| `labels` / `commitment_labels` | Etiquetas | N:N, sem efeito na cobrança |
| `commitments` | Compromissos | `recurrence_rule` (RRULE), `nag_profile`, `carry_over_policy`, `archived_at` |
| `occurrences` | Ocorrências | `due_at` (timestamptz), `timezone`, `state`, `current_step`; único em `(commitment_id, due_at)` |
| `deliveries` | Entregas | único em `(occurrence_id, step_index, channel_id)`, `correlation_id`, `provider_message_id`, `status` |
| `acks` | Reconhecimentos | `kind` (`completion`/`snooze`/`dismissal`), `source` (`web`/`channel`/`system`), `channel_id` nulo |
| `river_job` | fila | gerenciada pelas migrations do River |

**Estados de uma Ocorrência:**

```
pending ──▶ due ──▶ nagging ──┬──▶ done        (Conclusão)
                              ├──▶ snoozed ──▶ due   (Adiamento; a escada retoma, não reinicia)
                              ├──▶ dismissed  (Desistência)
                              └──▶ abandoned  (escada esgotada ou fora do Horizonte)
```

`done`, `dismissed` e `abandoned` são terminais. Nenhuma transição apaga linha: o
histórico de por que uma Ocorrência morreu é o dado mais valioso do sistema.

**Senhas e segredos.** Credenciais de Canais (token da Evolution, senha SMTP, futuros
tokens OAuth) são cifradas com AES-GCM usando uma chave de aplicação, nunca retornadas
pela API e nunca logadas — nem em nível debug.

## 5. Camadas do backend

```
backend/
├── cmd/
│   ├── api/            servidor HTTP
│   ├── worker/         runtime do River
│   └── admin/          criação de usuário, verificação de Canal, migrations
├── internal/
│   ├── domain/         entidades e regras puras — sem Gin, sem GORM, sem rede
│   │   ├── commitment/
│   │   ├── occurrence/ máquina de estados, escada de Degraus, Janela de Silêncio
│   │   └── channel/    a porta Channel
│   ├── app/            casos de uso; orquestra domínio + repositórios
│   ├── adapters/
│   │   ├── postgres/   repositórios GORM
│   │   ├── evolution/  adaptador WhatsApp
│   │   ├── smtp/       adaptador e-mail
│   │   └── http/       handlers Gin, DTOs, middleware, anotações Swagger
│   ├── jobs/           Materializador, Scanner, Entregador, Guardião
│   └── platform/       config, logging, crypto, relógio
└── migrations/
```

A regra que sustenta tudo: **`internal/domain` não importa nada de infraestrutura.** O
cálculo de qual Degrau está maduro, se a Janela de Silêncio está ativa e qual o próximo
vencimento de uma recorrência é função pura de `(estado, agora, fuso)`. É isso que torna
o motor testável sem Postgres, sem WhatsApp e sem esperar o relógio — o relógio é injetado
como interface.

### A porta `Channel`

```go
type Channel interface {
    Kind() string
    Send(ctx context.Context, msg Message) (ProviderID, error)
    HealthCheck(ctx context.Context) error
    SupportsInboundAck() bool   // false em todos os adaptadores do MVP
}
```

`SupportsInboundAck` existe desde já e retorna `false` em todo lugar. É o gancho para o
Reconhecimento pelo próprio Canal ([ADR-0006](./docs/adr/0006-reconhecimento-apenas-pelo-app-web-no-mvp.md)),
e toda Entrega já grava o `correlation_id` que tornará a correlação resposta→Ocorrência
possível — esse é o único dado do desenho que não poderia ser reconstruído
retroativamente.

## 6. Frontend

Vite + React + Tailwind, SPA servida por nginx, consumindo `/api/v1` com JWT em cookie
`HttpOnly` + `SameSite=Strict`.

Três telas bastam para o MVP:

1. **Hoje** — a tela inicial. O que vence hoje e o que está cobrando agora, com os três
   botões de Reconhecimento lado a lado. **Desistir tem o mesmo peso visual que Concluir**;
   escondê-lo faz o usuário concluir mentindo, e aí o sistema perde a capacidade de
   distinguir o que ele faz do que ele só diz que vai fazer.
2. **Compromissos** — CRUD, recorrência, Área, Perfil de Insistência, Etiquetas.
3. **Ajustes** — fuso, Janela de Silêncio, Canais e sua ordem de fallback.

Não há Kanban. O documento original define o produto pela negação do painel bonito
esquecido em duas semanas; construir o painel de cards seria contradizer a própria tese.

## 7. Autenticação

JWT de acesso curto (15 min) + refresh token rotativo persistido. Cadastro fechado por
padrão ([ADR-0001](./docs/adr/0001-self-hosted-inquilino-unico.md)); usuários são criados
via `cmd/admin`.

**Verificação de Canal é obrigatória antes da primeira Entrega real.** Um Canal só sai de
`verified_at IS NULL` depois que um código enviado para ele é devolvido no app. Sem isso,
qualquer um digita o número de um terceiro e o sistema vira ferramenta de spam com
insistência configurável.

## 8. Observabilidade

Log estruturado em JSON (`log/slog`), com `user_id`, `occurrence_id` e `correlation_id`
propagados por contexto. Nenhum conteúdo de mensagem e nenhum segredo vai para o log.

Quatro sinais que precisam de alerta, porque falham em silêncio:

- **Scanner sem execução há mais de 5 minutos.** É o produto inteiro parado.
- **Sessão da EvolutionAPI desconectada.** Alerta por e-mail, que não depende dela.
- **Taxa de falha de Entrega por Canal** acima do limiar.
- **Taxa de Abandono** — o gatilho documentado em
  [ADR-0006](./docs/adr/0006-reconhecimento-apenas-pelo-app-web-no-mvp.md) para
  implementar o Reconhecimento por Canal.

## 9. Deploy

Dois arquivos compose, sem herança entre eles — a duplicação é preferível ao `extends`
que ninguém entende seis meses depois.

**`docker-compose.local.yml`** — postgres, api, worker, web, evolution. Portas expostas
(`5432`, `8080`, `5173`), sem TLS, sem labels Traefik, logs em texto legível, hot reload.

**`docker-compose.prod.yml`** — os mesmos serviços, **sem nenhuma porta publicada**. Todo
tráfego entra por Traefik via labels, com TLS automático. Logs JSON com rotação,
`restart: unless-stopped`, health checks e limites de recurso em todos os serviços.
Volumes nomeados para dados do Postgres e sessão da Evolution — a sessão da Evolution é
estado real: perdê-la significa reparear o QR code.

Segredos por variáveis de ambiente a partir de um `.env` fora do versionamento;
`.env.example` documenta cada uma.

## 10. Estado atual

Nada foi implementado. O repositório tem `go.mod` e esta documentação. A ordem sugerida de
construção está no [`README.md`](./README.md).
