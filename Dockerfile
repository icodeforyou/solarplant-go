FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ./bin/solarplant ./main.go

FROM alpine:3.21 AS runner
RUN apk add --no-cache tzdata
WORKDIR /app
COPY --from=builder /build/bin/solarplant solarplant
COPY www/templates www/templates/
COPY www/static www/static/
ENTRYPOINT ["/app/solarplant"]
