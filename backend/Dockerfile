FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o jd-server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
RUN mkdir -p /app/data
COPY --from=builder /build/jd-server .
ENV DB_PATH=/app/data/app.db
EXPOSE 8080
CMD ["./jd-server"]
