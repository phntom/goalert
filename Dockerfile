FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum src ./
RUN go build -o /app/alert
ENV CGO_ENABLED 0
RUN go test

FROM alpine:3.14
WORKDIR /app
COPY --from=builder /app/alert ./
ENV AUTH_TOKEN=""
ENV CHAT_DOMAIN="https://kix.co.il"
ENV TEAM_NAME="nix"
ENV CHANNEL_NAME="alerts"

CMD ["/app/alert"]
