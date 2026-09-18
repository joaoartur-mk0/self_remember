# Self-hosted de inquilino único, sem `tenant_id`

O Self Remember é implantado por quem o usa, em uma instância própria, para si e
eventualmente para um punhado de pessoas de confiança. Existe tabela `users` e todo dado
é escopado por `user_id`, mas **não** existe conceito de inquilino: nada de `tenant_id`,
billing, provisionamento automático de instância WhatsApp por usuário ou verificação de
e-mail obrigatória no cadastro.

Consideramos modelar `tenant_id` desde já "para não migrar depois". Rejeitamos: o risco
real deste produto não é a escala, é descobrir se o motor de cobrança realmente muda o
comportamento de alguém. Pagar complexidade de multi-inquilino antes disso é otimizar a
parte que talvez nunca precise existir. Se o SaaS vier, `user_id` já isola todas as
consultas e a migração é uma coluna e um escopo no repositório — caro, mas previsível.

## Consequências

- O banco confia no `user_id` do token; não há Row Level Security no Postgres.
- Credenciais de terceiros (token da Evolution, SMTP) podem ser de instância, não
  necessariamente por usuário.
- O cadastro é fechado por padrão (`SELF_REMEMBER_ALLOW_SIGNUP=false`); usuários são
  criados por um comando administrativo.
