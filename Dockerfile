FROM golang:alpine3.18 AS build
WORKDIR /build
COPY . . 
RUN apk add make; make build

FROM alpine:3.18
WORKDIR /app
COPY --from=build /build/firefly-iii-bank-sync /app/firefly-iii-bank-sync
CMD ["/app/firefly-iii-bank-sync"]
