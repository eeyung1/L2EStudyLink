FROM golang:1.21-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o L2EStudyLink .

EXPOSE 8080

CMD ["./L2EStudyLink"]
