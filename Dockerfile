FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum /app/
COPY internal /app/internal/
COPY cmd /app/cmd/
#RUN ls && false
RUN go mod tidy
RUN go build -o alert /app/cmd/goalert-bot
ENV CGO_ENABLED 0
RUN go test ./...

FROM alpine:3.14
WORKDIR /app
COPY --from=builder /app/alert ./
ENV AUTH_TOKEN=""
ENV CHAT_DOMAIN="https://kix.co.il"
EXPOSE 3000

CMD ["/app/alert"]
