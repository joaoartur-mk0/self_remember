# Changelog

Formato baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/);
versionamento conforme [SemVer](https://semver.org/lang/pt-BR/).

## [Não lançado]

### Adicionado

- `cmd/api`: esqueleto do processo — config via `SELF_REMEMBER_*`, log estruturado
  em JSON e `GET /healthz` (F1 da Fundação).
- Migration `0001_users` (goose): tabelas `users` e `user_settings`. Pool GORM
  aberto sobre a conexão pgx já migrada, sem `AutoMigrate` (F2 da Fundação).
- ADR-0008: revogação de toda a sessão do usuário no reuso de um refresh token
  já rotacionado.
- Autenticação: `POST /api/v1/auth/login` e `/auth/refresh`, JWT de acesso
  (15 min) em cookie `HttpOnly` + `SameSite=Strict`, refresh token rotativo
  persistido hasheado em `refresh_tokens` (migration `0002`) (F3 da Fundação).
- `cmd/admin create-user`: cria o primeiro usuário, com senha lida de stdin
  (F4 da Fundação).
- Swagger em `/swagger/index.html` para os endpoints de autenticação, gerado
  via swaggo/swag (F5 da Fundação — última etapa).
- Porta `Channel` em domínio puro (`internal/domain/channel`), com `Kind`
  tipado e `SupportsInboundAck` (ADR-0006).
- Entidade `Commitment` (`internal/domain/commitment`) com `NagProfile`
  (ADR-0004), `CarryOverPolicy` e validação.
- Recorrência de Compromisso com fuso (`NextOccurrenceAfter`, via
  `teambition/rrule-go`), testada atravessando uma virada de horário de
  verão real (America/New_York, 2024-03-10).
- Máquina de estados da Ocorrência (`internal/domain/occurrence`): transições
  literais do diagrama em `ARCHITECTURE.md` §4, com `Snooze`/`Resume`
  preservando o Degrau atual.
- Escada de insistência (`DelayForStep`, `ShouldEscalate`) para os três
  Perfis do ADR-0004, incluindo o Implacável sem teto.
- Janela de Silêncio (`QuietHours`) e Horizonte de Relevância (7 dias) em
  domínio puro — encerra a Fase 2 (Domínio puro): tudo com relógio injetado,
  coberto por testes, sem nenhuma linha de rede.
- Correção de domínio: `Occurrence.Abandon` passa a valer também a partir de
  `due`, não só `nagging` — cobre o caso do ARCHITECTURE.md §3.4 em que uma
  Ocorrência sai do Horizonte de Relevância sem nunca ter sido cobrada.
  Diagrama de estados em `ARCHITECTURE.md` §4 atualizado junto.
- Migrations `0003`–`0007` (goose): `areas`, `channels`, `commitments`,
  `occurrences`, `deliveries` — idempotência do Materializador via
  `UNIQUE(commitment_id, due_at)` e do Entregador via
  `UNIQUE(occurrence_id, step_index, channel_id)`.
- `cmd/worker`: motor de cobrança sobre River (ADR-0007) — Materializador
  (`1h`), Scanner (`1m`, único produtor de jobs de Entrega, ADR-0003),
  Entregador (idempotente, com cadeia de fallback) e Guardião (`5m`, escada
  esgotada e Horizonte de Relevância). Canal falso de log
  (`internal/adapters/logchannel`) prova o motor de ponta a ponta sem
  nenhum adaptador real ainda — encerra a Fase 3 (Motor).
- Canal de e-mail real (`internal/adapters/smtp`, via `wneessen/go-mail`),
  TLS oportunista para não depender de STARTTLS em ambiente local.
- Canal de WhatsApp real (`internal/adapters/evolution`, HTTP direto contra a
  EvolutionAPI): teto de mensagens por minuto com espaçamento aleatório e
  `HealthCheck` de sessão, ambos exigidos por ADR-0005.
- `internal/jobs/verificador_canal.go`: `PeriodicJob` de 1 minuto dedicado ao
  health check da EvolutionAPI (cadência própria do ADR-0005, mais fina que
  o ciclo de 5 minutos do resto do Guardião), com alerta por e-mail na
  transição para sessão caída. Remove o canal falso de log — encerra a
  Fase 4 (Canais reais).
- `GET/POST /api/v1/areas` e `GET/POST /api/v1/labels`: início da Fase 5
  (API e frontend). `AreaModel` ganha `color`, prometido desde a migration
  `0003_areas.sql` para esta fase; Janela de Silêncio própria por Área fica
  para quando existir lógica de override no domínio. Nomes duplicados por
  usuário são rejeitados com 409 (`postgres.ErrDuplicateName`,
  `UNIQUE(user_id, name)` — migrations `0010`/já existente em `labels` desde
  a `0008`). `Update`/`Delete` ficam para um commit à parte: exigem decidir
  a política de exclusão com Compromissos já associados.
- Tabela `acks` (migration `0012`) e `AckRepository`: log de auditoria de
  Reconhecimentos — quando e como o usuário reagiu a uma Ocorrência, "o dado
  mais valioso do sistema" (CONTEXT.md). Só `INSERT`, nunca `UPDATE`/`DELETE`.
- `AreaRepository.FindByIDForUser`: busca escopada por usuário, no padrão de
  `ChannelRepository`, reintroduzida para o caso de uso de Compromisso validar
  posse de Área.
- Ajustes: `GET/PUT /api/v1/settings` sobre `user_settings` (migration `0001`,
  sem migration nova). `PUT` valida fuso IANA (`time.LoadLocation`) antes de
  gravar; Janela de Silêncio aceita `"HH:MM"` e é persistida como `TIME`.
- Compromissos: `POST/GET /api/v1/commitments` e
  `POST /api/v1/commitments/:id/archive`. `CreateUseCase` confirma posse de
  Área e Etiquetas antes de persistir — posse recusada devolve 400, não 404,
  para não vazar existência de recursos de outro usuário. Edição (`PUT`) fica
  fora desta fase: depende de regenerar Ocorrências futuras (ADR-0002 ainda
  não existe).
- Ocorrências: `GET /api/v1/occurrences/today` (vencimento calculado no fuso
  do usuário) e `POST /api/v1/occurrences/:id/{complete,dismiss,snooze}`.
  `AckUseCase` aplica a transição de domínio, persiste, e grava um `Ack` de
  auditoria (`source: "web"`). Transições inválidas (`*TransitionError`) e
  Adiamento para um vencimento no passado (novo
  `occurrence.ErrSnoozeMustBeAfterCurrent`) devolvem 409, não 500.
- `cmd/admin create-user` exige `--timezone` (validado como IANA) e cria
  `users`/`user_settings` na mesma transação — um usuário nunca existe sem
  fuso, encerrando a lacuna que a Fase 5 (API mínima para o frontend existir)
  precisava fechar.
- `frontend/`: Vite + React + Tailwind v4 (`@tailwindcss/vite`), react-router.
  Três telas — Hoje (Reconhecimento com os três botões em peso visual igual,
  ARCHITECTURE.md §6), Compromissos (criação + Arquivar) e Ajustes (fuso via
  `Intl.supportedValuesOf`, Janela de Silêncio). `src/lib/api.ts`:
  `credentials: 'include'` sempre, uma única tentativa de `/auth/refresh` em
  401 antes de redirecionar para `/login`. Sem UI kit.
- `docker-compose.local.yml` e `backend/Dockerfile`/`frontend/Dockerfile`:
  composição local com Postgres, `api`, `worker`, `web` (nginx fazendo proxy
  reverso de `/api/` para `api:8080` — obrigatório para o cookie
  `SameSite=Strict` do backend chegar) e `evolution`. Desvio do plano
  original: a EvolutionAPI não aceita `DATABASE_PROVIDER=sqlite` (só
  `postgresql`/`mysql`, confirmado na documentação oficial) — ela usa uma
  segunda base no mesmo Postgres do compose (`docker/postgres-init`), não
  SQLite interno.
- Canais: `POST/GET/DELETE /api/v1/channels` e
  `POST /api/v1/channels/:id/{verify/request,verify/confirm}`. Verificação por
  código de 6 dígitos, hasheado (sha256) com expiração de 15 min, enviado pelo
  próprio Canal a verificar (SMTP ou EvolutionAPI) — um Canal só entra na
  cadeia de fallback depois de confirmado (ARCHITECTURE.md §7). Novo Canal
  sempre entra ao fim da cadeia; reordenar fica para uma tela futura.
  `cmd/api` passa a carregar config de SMTP/Evolution além do `cmd/worker`
  (antes só quem enviava mensagens de verdade lia essas variáveis).
- `GET /api/v1/whatsapp-instance/{status,qrcode}`: painel de status e
  pareamento por QR code da única instância WhatsApp desta instalação — não
  há provisionamento por usuário (ADR-0001 já rejeita isso explicitamente).
- Correção: `Request()` da verificação de Canal misturava falha de
  persistência (nosso banco, 500) com falha de envio (SMTP/EvolutionAPI fora
  do ar) — as duas viravam 502. Novo `ErrSendFailed` isola só o segundo caso.
  `.env.example` atualizado para refletir que `cmd/api` também lê essas
  variáveis agora. Achado pelo `/code-review` dos dois commits anteriores.
- Compromissos: recorrência por presets (Não repete/Diário/Semanal/Mensal)
  substitui o campo de texto RRULE cru no formulário — `BYDAY`/`BYMONTHDAY`
  seguem o `DTSTART` por padrão (RFC 5545). Perfil de Insistência e Política
  de Acúmulo do formulário vêm pré-selecionados de `GET /settings`.
- Ajustes: migration `0013` adiciona `default_nag_profile`/
  `default_carry_over_policy` a `user_settings` — só pré-selecionam o
  formulário de criação de Compromisso, sem efeito em Compromissos já
  criados nem no motor de cobrança. `cmd/admin create-user` grava os mesmos
  valores do `DEFAULT` da coluna explicitamente (um `Upsert` com zero-value
  sobrescreveria o default do banco).
- Correção: o painel do QR code buscava o código pela primeira vez só depois
  de 5s (`setInterval` só dispara no primeiro tick após o intervalo) — a
  tela ficava em branco até lá. Achado pelo `/code-review`.
- `.dockerignore` na raiz do repo: o contexto de build do `backend/Dockerfile`
  é a raiz (não `backend/`), então precisava excluir `.git`,
  `frontend/node_modules` e `frontend/dist` daqui, não de dentro de `backend/`.
- Estágio `admin` em `backend/Dockerfile` e serviço `admin` (perfil `tools`,
  fora do `up` padrão) em `docker-compose.local.yml`: não existia jeito de
  rodar `cmd/admin create-user` — a única forma de criar o primeiro usuário —
  sem Go instalado no host. `docker compose --profile tools run --rm admin
  create-user ...` fecha isso. README atualizado com o comando (também
  corrige a seção "Rodando localmente", que ainda dizia "ainda não há o que
  rodar").
- Healthcheck no serviço `evolution` do compose local: no primeiro boot ela
  roda ~42 migrations do Prisma antes de aceitar conexão (30-60s), e
  `worker`/`api` chamavam `http://evolution:8080` nesse meio-tempo — os 502 e
  "connection refused" nos logs são o comportamento correto de degradação
  graciosa (ADR-0005), não um defeito. O healthcheck existe só para
  visibilidade em `docker compose ps`; `worker`/`api` continuam sem esperar
  por ele de propósito, porque WhatsApp é um Canal com fallback, não uma
  dependência dura do resto do sistema.
- Correção: causa real do "EvolutionAPI inacessível" persistente (não só o
  warm-up). Uma instalação nova da EvolutionAPI não vem com nenhuma
  instância, e nada no código criava a instância desta instalação — todo
  `ConnectionState`/`Connect`/`Send` batia num 404 traduzido para erro
  genérico de "inacessível". Novo `ErrInstanceNotFound` distingue "instância
  não existe" de falha de conectividade de verdade; `EnsureInstance`
  provisiona sob demanda (`POST /instance/create`) a partir de `Connect` e
  `Send` — ações deliberadas de usar o WhatsApp agora — nunca a partir de
  `HealthCheck`, que só deve reportar o que existe. `GET
  /whatsapp-instance/status` devolve `state: "not_provisioned"` (200, não
  502) nesse caso; o painel mostra "nunca configurado" em vez de erro.
- Correção: QR code nunca aparecia mesmo com a instância provisionada e o
  estado em `"connecting"`. O painel chamava `GET /whatsapp-instance/qrcode`
  (→ `/instance/connect`) a cada 5s indefinidamente — a EvolutionAPI v2.1.1
  entra num loop de reconexão e nunca produz QR code se esse endpoint for
  chamado de novo antes do handshake anterior terminar (bug documentado
  upstream, issues #2385/#2437). `WhatsAppInstancePanel` agora só pede o QR
  uma vez por sessão de pareamento (auto na primeira vez que o estado não é
  `"open"`); um novo pedido depois disso é sempre um clique explícito no
  botão "Gerar novo QR code", nunca automático. O polling de `/status`
  continua a cada 5s — é leitura, não afeta a sessão.
- Imagem da `evolution` no compose local atualizada de `v2.1.1` para
  `v2.3.7` (estável mais recente no momento), na esperança de que o bug de
  reconexão acima também esteja corrigido lá. Não testado de ponta a ponta
  neste ambiente (sem acesso a Docker); o fix do polling acima é a correção
  que não depende dessa troca de versão ter funcionado.
- `AreasSection`: cadastro de Áreas embutido na própria tela de Compromissos
  (nome + cor), não em Ajustes — Área é pré-requisito para criar um
  Compromisso (o `<select>` do formulário não tinha de onde vir nenhuma
  opção antes disso) e fica no lugar onde é usada. Lista com a cor de cada
  Área já cadastrada; `Compromissos` repassa `onChange` para manter seu
  próprio `<select>` sincronizado sem cada um buscar `/areas` por conta
  própria.
- Formulário de Compromissos: o campo único "Vencimento inicial" + presets de
  RRULE sem parâmetro deu lugar a campos por tipo de recorrência — Pontual
  pede data e hora; Diário só a hora; Semanal, os dias da semana (múltiplos)
  e a hora; Mensal, o dia do mês e a hora. Sem mudança no backend: a
  expansão (`rrule-go`) já aceita RRULE livre, então o frontend só passou a
  montar `BYDAY=MO,FR` / `BYMONTHDAY=15` explícitos em vez de deixar
  implícito no `DTSTART`. Dois testes novos
  (`TestNextOccurrenceAfter_WeeklyByDay`/`_MonthlyByMonthDay`) travam que
  isso funciona mesmo com `RecurrenceStart` ("hoje", que é só a âncora) num
  dia diferente do pedido pela regra.
- Correção: Compromisso com vencimento próximo não aparecia na tela Hoje nem
  disparava — podia até nunca materializar Ocorrência nenhuma. Causa: o
  backend reinterpreta os dígitos de `recurrence_start` como hora de parede
  no fuso de Ajustes (`wallClockIn`, ARCHITECTURE.md §3.1), ignorando o
  deslocamento que a string carrega; mas o frontend montava
  `recurrence_start` com `new Date(...).toISOString()`, que converte pelo
  fuso do **navegador** — corrompendo os dígitos sempre que esse fuso não é
  UTC+0. Se o desvio empurrasse a hora pro passado, `NextOccurrenceAfter`
  devolvia `ok=false` e nenhuma Ocorrência era criada, silenciosamente. Bug
  pré-existente aos presets de recorrência desta mesma fase, não introduzido
  por eles. Os quatro modos de recorrência agora montam `recurrence_start` a
  partir dos dígitos literais escolhidos pelo usuário, sem nenhuma conversão
  de fuso no meio. O Adiamento (`Hoje.tsx`) não tinha esse problema — `due_at`
  aí é um instante real comparado diretamente (`Occurrence.Snooze`), sem
  `wallClockIn`.

- Correção: Compromisso com vencimento próximo materializava corretamente
  (fuso já corrigido antes) mas ainda podia ficar até 1h sem aparecer em
  lugar nenhum. Causa: o Materializador só roda de hora em hora
  (`PeriodicInterval(1h)`) — `RunOnStart: true` só dispara no boot do
  worker, não a cada Compromisso novo, então um Compromisso criado entre
  dois ticks esperava até o próximo. Lógica de materialização de um único
  Compromisso extraída de `jobs.MaterializadorWorker` para
  `commitmentapp.MaterializeUseCase` (idempotente via
  `UNIQUE(commitment_id, due_at)`, ADR-0002 — chamar de novo nunca duplica),
  e `CreateUseCase.Create` passa a chamá-la uma vez, eager, logo após criar
  o Compromisso. Best-effort: se não houver `user_settings` ainda (contas
  anteriores a `cmd/admin` exigir `--timezone`), a criação não falha por
  isso — a varredura periódica continua sendo quem garante a materialização
  de verdade, isto só evita a espera de até 1h à toa.

- Compromisso ganha `description`, campo principal junto de título e Área
  (migration `0014`, `NOT NULL DEFAULT ''` na coluna — Compromissos
  existentes não quebram; `commitment.ErrDescriptionRequired` exige um
  valor não vazio em Compromissos novos, mesma regra do título). A mensagem
  de Cobrança (`internal/jobs/entregador.go`) muda de "Título venceu em
  DD/MM HH:MM" para título em negrito no topo (reconhecimento), Área e
  descrição abaixo (o que é), vencimento por último —
  `OccurrenceSummary`/`FindSummary` ganham `AreaName`/`Description` via
  join com `areas` para isso. Formulário e lista de Compromissos no
  frontend atualizados.

- CRUD de Área completo: `PUT/DELETE /api/v1/areas/:id`, escopados por
  usuário (`AreaRepository.UpdateForUser`/`DeleteForUser`). Remover uma Área
  com Compromissos associados devolve 409 (`postgres.ErrInUse`, tradução da
  violação de FK 23503 — `commitments.area_id` é `NOT NULL REFERENCES
  areas(id)` sem `ON DELETE`, migration 0005) em vez de deixar o Postgres
  estourar um 500 cru; a decisão de produto que ficou em aberto no commit
  original do CRUD de Área (o que fazer com Compromissos de uma Área
  removida) se resolve assim: bloquear, nunca apagar ou desvincular em
  cascata. `AreasSection` ganha edição e remoção inline.
- `CONTEXT.md`: glossário normativo do domínio (Compromisso, Ocorrência, Cobrança,
  Entrega, Canal, Perfil de Insistência, Reconhecimento, Abandono).
- `ARCHITECTURE.md`: estrutura do sistema, motor de cobrança, modelo de dados e deploy.
- `README.md`: visão do produto, stack e ordem sugerida de construção.
- ADRs 0001–0007 registrando as decisões estruturais iniciais.
