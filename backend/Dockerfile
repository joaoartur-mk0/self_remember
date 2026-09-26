# Contexto de build: raiz do repo (go.mod está lá, backend/ é só um
# subdiretório do módulo self-remember). docker-compose.local.yml usa
# `context: .` + `dockerfile: backend/Dockerfile`.
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY backend/ backend/

# Migrations e assets do River vêm embutidos via go:embed — sem COPY extra.
RUN CGO_ENABLED=0 go build -o /out/api ./backend/cmd/api
RUN CGO_ENABLED=0 go build -o /out/worker ./backend/cmd/worker

FROM gcr.io/distroless/static-debian12 AS api
COPY --from=builder /out/api /api
ENTRYPOINT ["/api"]

FROM gcr.io/distroless/static-debian12 AS worker
COPY --from=builder /out/worker /worker
ENTRYPOINT ["/worker"]
