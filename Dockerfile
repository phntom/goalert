FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
WORKDIR /app
ARG TARGETOS TARGETARCH
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY internal ./internal/
COPY cmd ./cmd/
RUN go vet ./... && go test ./...
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/goalert ./cmd/goalert

FROM alpine:3.21
RUN apk --no-cache add tzdata ca-certificates
WORKDIR /app
COPY --from=builder /out/goalert ./goalert
ENV CHAT_DOMAIN="https://kix.co.il"
EXPOSE 3000
USER nobody
ENTRYPOINT ["/app/goalert"]
