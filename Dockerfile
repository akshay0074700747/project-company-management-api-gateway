FROM golang:1.21.5-alpine 

RUN apt-get update && apt-get install -y curl libcurl-dev

WORKDIR /app

COPY . .

RUN go mod download

WORKDIR /app/cmd

RUN go build -o api-gateway

FROM busybox:latest

WORKDIR /api-gateway

COPY --from=build /app/cmd/api-gateway .

COPY --from=build /app/cmd/.env .

EXPOSE 50000

CMD [ "./api-gateway" ]