FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY cmd/webhook-stub/go.mod ./
COPY cmd/webhook-stub/*.go ./

RUN go build -o webhook

FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/webhook .
EXPOSE 9090

CMD ["./webhook"]