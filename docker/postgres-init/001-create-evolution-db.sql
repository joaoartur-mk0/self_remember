-- A EvolutionAPI (serviço "evolution" no docker-compose.local.yml) só aceita
-- DATABASE_PROVIDER=postgresql ou mysql — sqlite não existe como opção na
-- versão atual (confirmado na documentação oficial), então ela usa uma
-- segunda base no mesmo Postgres do compose, isolada da base da aplicação.
-- Scripts em /docker-entrypoint-initdb.d só rodam na primeira inicialização
-- do volume (Postgres vazio) — não são migrations, não repetem.
CREATE DATABASE evolution;
