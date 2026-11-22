# Keep the container toolchain aligned with go.mod and CI.
FROM golang:1.26.1-alpine3.23

ENV GOTOOLCHAIN=local
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN sh scripts/build.sh

EXPOSE 8080
CMD ["sh", "./start.sh"]
