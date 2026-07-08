FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/whooper .

FROM alpine:3.21

RUN apk add --no-cache wget

COPY --from=build /out/whooper /usr/local/bin/whooper

ENV WHOOPER_HOME=/data
WORKDIR /data

ENTRYPOINT ["whooper"]
