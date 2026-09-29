# Contexto de build: raiz do repo (go.mod está lá, backend/ é só um
# subdiretório do módulo self-remember). docker-compose.local.yml usa
# `context: .` + `dockerfile: backend/Dockerfile`.
FROM golang:1.26-alpine AS builder

# A imagem trava GOTOOLCHAIN=local, então uma patch de go.mod mais nova que a
# imagem base (ex.: go.mod pede 1.26.5, a imagem tem 1.26.4) falha o build em
# vez de baixar o toolchain certo — auto restaura o comportamento padrão do
# Go 1.21+.
ENV GOTOOLCHAIN=auto

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY backend/ backend/

# Migrations e assets do River vêm embutidos via go:embed — sem COPY extra.
RUN CGO_ENABLED=0 go build -o /out/api ./backend/cmd/api
RUN CGO_ENABLED=0 go build -o /out/worker ./backend/cmd/worker
RUN CGO_ENABLED=0 go build -o /out/admin ./backend/cmd/admin

FROM gcr.io/distroless/static-debian12 AS api
COPY --from=builder /out/api /api
ENTRYPOINT ["/api"]

FROM gcr.io/distroless/static-debian12 AS worker
COPY --from=builder /out/worker /worker
ENTRYPOINT ["/worker"]

# Não sobe como serviço de longa duração — é para `docker compose run --rm
# admin <comando>` (cadastro fechado por padrão, ADR-0001, não tem HTTP
# nenhum para expor). distroless/static não tem shell, então a leitura de
# senha de readPasswordFromStdin ainda funciona: é `os.Stdin` puro, sem TTY
# nem shell envolvido.
FROM gcr.io/distroless/static-debian12 AS admin
COPY --from=builder /out/admin /admin
ENTRYPOINT ["/admin"]
