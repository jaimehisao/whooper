FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/whooper .

FROM alpine:3.21

RUN adduser -D -u 10001 whooper \
	&& apk add --no-cache wget \
	&& mkdir -p /data \
	&& chown whooper:whooper /data

COPY --from=build /out/whooper /usr/local/bin/whooper

ENV WHOOPER_HOME=/data
WORKDIR /data
USER whooper

# Default for container / GHA service deployments. Compose may override the
# command. Bind 0.0.0.0 so host port maps reach the process inside the container.
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
	CMD wget -qO- http://127.0.0.1:9464/healthz || exit 1

ENTRYPOINT ["whooper"]
CMD ["serve", "--addr", "0.0.0.0:9464"]
