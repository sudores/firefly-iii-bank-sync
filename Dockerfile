FROM golang:alpine3.18 AS build
WORKDIR /build
COPY . . 
RUN apk add make; make build

FROM alpine:3.18
WORKDIR /app
COPY . .
CMD ['/app/firefly-iii-bank-sync']
