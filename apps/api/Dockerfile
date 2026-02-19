FROM oven/bun:1.2.13 AS bun-builder

WORKDIR /repo
COPY package.json bun.lock turbo.json ./
COPY packages ./packages
COPY apps/web/package.json ./apps/web/
COPY apps/api ./apps/api

RUN bun install --frozen-lockfile --ignore-scripts
RUN bun run openapi:generate
RUN bun run emails:generate

FROM golang:1.24.5 AS go-builder

WORKDIR /repo
COPY apps/api/go.mod apps/api/go.sum ./apps/api/

WORKDIR /repo/apps/api
RUN go mod download

COPY apps/api /repo/apps/api
COPY --from=bun-builder /repo/apps/api/static /repo/apps/api/static
COPY --from=bun-builder /repo/apps/api/templates /repo/apps/api/templates

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrate ./cmd/migrate

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=go-builder /out/api /app/api
COPY --from=go-builder /out/migrate /app/migrate
COPY --from=go-builder /repo/apps/api/templates /app/templates
COPY --from=go-builder /repo/apps/api/static /app/static

EXPOSE 8080
USER app

CMD ["/app/api"]
