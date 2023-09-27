FROM golang:alpine3.18 AS build
WORKDIR /build
COPY . . 
RUN apk add make; make build

FROM alpine:3.18
WORKDIR /app
COPY --from=build /build/app /app/app
CMD ["/app/app"]
