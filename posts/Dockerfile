FROM golang:1.24-alpine

ARG project=posts

WORKDIR /workspace

COPY go.work go.work.sum ./

COPY ./api_gateway/go.mod ./api_gateway/go.sum ./api_gateway/
COPY ./passport/go.mod ./passport/go.sum ./passport/
COPY ./posts/go.mod ./posts/go.sum ./posts/
COPY ./proto/ ./proto/

RUN go mod download


COPY ${project}/ ./${project}/
WORKDIR /workspace/${project}

RUN go build -o main .

EXPOSE 8080

CMD ["./main"]
