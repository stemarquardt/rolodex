FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1020
RUN templ generate
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.20
WORKDIR /app

COPY --from=builder /out/server ./server
COPY static ./static

ENV DB_PATH=/data/people.db
ENV STATIC_DIR=/app/static
ENV ADDR=:8080

EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["./server"]
