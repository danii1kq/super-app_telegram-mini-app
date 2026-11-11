# Build the web bundle, then the Go binaries.
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -o /out/worker ./cmd/worker \
 && CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate

FROM alpine:3.20
RUN adduser -D -u 10001 srunal
WORKDIR /app
COPY --from=backend /out/ /app/bin/
COPY --from=web /app/web/dist /app/web
COPY db/migrations /app/db/migrations
USER srunal
EXPOSE 8080
ENTRYPOINT ["/app/bin/api"]
