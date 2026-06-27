FROM golang:1.26.1 AS go-builder
WORKDIR /app/allocator

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /app/terminal .
