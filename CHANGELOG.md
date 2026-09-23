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

- `CONTEXT.md`: glossário normativo do domínio (Compromisso, Ocorrência, Cobrança,
  Entrega, Canal, Perfil de Insistência, Reconhecimento, Abandono).
- `ARCHITECTURE.md`: estrutura do sistema, motor de cobrança, modelo de dados e deploy.
- `README.md`: visão do produto, stack e ordem sugerida de construção.
- ADRs 0001–0007 registrando as decisões estruturais iniciais.
