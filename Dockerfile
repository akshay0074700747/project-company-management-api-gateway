FROM golang:1.21.5-alpine AS build

WORKDIR /app

COPY . .

RUN go mod download

RUN go mod tidy

RUN go get -u github.com/confluentinc/confluent-kafka-go/kafka

WORKDIR /app/cmd

RUN go build -o api-gateway

FROM busybox:latest

WORKDIR /api-gateway

COPY --from=build /app/cmd/api-gateway .

COPY --from=build /app/cmd/.env .

EXPOSE 50000

CMD [ "./api-gateway" ]