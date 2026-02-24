FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/http

FROM alpine:3.21

WORKDIR /app
COPY --from=builder /out/server /app/server

EXPOSE 8080

CMD ["/app/server"]
