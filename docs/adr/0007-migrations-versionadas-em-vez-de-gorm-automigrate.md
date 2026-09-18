# Migrations versionadas com goose; `AutoMigrate` proibido

O schema é gerenciado por arquivos SQL versionados em `backend/migrations/`, aplicados
pelo goose. `gorm.AutoMigrate` não é chamado em lugar nenhum, em nenhum ambiente,
inclusive local.

`AutoMigrate` é o caminho óbvio de quem usa GORM, então a proibição precisa de motivo
registrado: ele não remove colunas, não renomeia, não altera tipos de forma segura, não
tem ordem determinística e não tem reversão. Ele funciona bem exatamente até o primeiro
dado real existir, e a partir daí o custo de sair dele cresce monotonicamente. Este
sistema guarda o histórico de compromissos pessoais de alguém — não é um schema que se
possa recriar do zero quando der errado.

Usar também em ambiente local é deliberado: é a única forma de garantir que as migrations
estão corretas antes de chegarem em produção.

## Consequências

- O River tem as próprias migrations (`river migrate-up`), aplicadas separadamente das
  nossas. O entrypoint do container roda as duas, nessa ordem.
- As structs GORM e o SQL podem divergir silenciosamente. Um teste de integração compara o
  schema aplicado com o esperado pelos modelos a cada CI.
