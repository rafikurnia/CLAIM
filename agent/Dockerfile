# syntax=docker/dockerfile:1
FROM golang:1.18.7-alpine3.15 as builder
WORKDIR /app
COPY . ./
RUN apk add --no-cache git && CGO_ENABLED=0 GOOS=linux go build -a -o app .

FROM alpine:3.15
WORKDIR /app
COPY --from=builder /app/app ./app
COPY curlt /usr/bin/curlt
RUN apk --no-cache add curl && chmod +x /usr/bin/curlt
CMD ["./app"]
