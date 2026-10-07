# --- UI ---
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- Go binary ---
FROM golang:1.22-alpine AS gobuild
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

# --- Runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates python3 \
  && ln -sf python3 /usr/bin/python
WORKDIR /app
COPY --from=gobuild /out/server /app/server
COPY --from=web /web/dist /app/web/dist
EXPOSE 8759
ENV HTTP_ADDR=:8759 \
    JUDGE_BACKEND=auto \
    PYTHON_BIN=python3
USER nobody
CMD ["/app/server"]
