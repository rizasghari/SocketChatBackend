FROM golang:latest

WORKDIR /app

COPY . .

RUN go get -d -v ./...

RUN go build -o socket-chat .

EXPOSE 8000

CMD ["./socket-chat"]