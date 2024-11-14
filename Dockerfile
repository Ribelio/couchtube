FROM golang:1.22.3-alpine AS builder

RUN apk update && apk add --no-cache sqlite

RUN mkdir -p /app/data

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o couchtube cmd/main.go

# final stage with only the built binary and necessary files
FROM alpine:latest
RUN apk add --no-cache sqlite curl
RUN mkdir -p /app/data
WORKDIR /app
COPY --from=builder /app/couchtube .
COPY static /app/static
EXPOSE 8363
CMD ["./couchtube"]
