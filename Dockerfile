FROM golang:1.22.5 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o kubedeploy main.go


FROM alpine:latest

WORKDIR /root

COPY --from=builder /app/kubedeploy .

EXPOSE 8080

CMD ["./kubedeploy"]
