FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum src ./
RUN go build -o /app/alert

FROM alpine:3.13
WORKDIR /app
COPY --from=builder /app/alert ./
ENV AUTH_TOKEN=""
ENV CHAT_DOMAIN="https://kix.co.il"
ENV TEAM_NAME="nix"
ENV CHANNEL_NAME="alerts"

CMD ["/app/alert"]
