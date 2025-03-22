FROM golang:1.21 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ai-proxy ./cmd/server

FROM alpine:3.18
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /app/ai-proxy /app/ai-proxy
COPY config/config.yaml /app/config/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/ai-proxy", "--config", "/app/config/config.yaml"] 