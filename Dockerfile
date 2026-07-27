# Multi-stage build for rackwire
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/rackwire ./cmd/rackwire

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/rackwire /app/rackwire
COPY data/rack.json /app/data/rack.json
ENV ADDR=:3040
ENV DATA_PATH=/app/data/rack.json
EXPOSE 3040
VOLUME ["/app/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:3040/api/health >/dev/null || exit 1
CMD ["/app/rackwire"]
