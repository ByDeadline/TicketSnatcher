FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o ticketsnatcher .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/ticketsnatcher .
RUN apk --no-cache add curl
CMD ["./ticketsnatcher"]