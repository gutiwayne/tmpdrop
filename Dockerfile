# --- build stage ----------------------------------------------------------
FROM golang:1.27.3-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/tmpdrop ./cmd/tmpdrop

# --- runtime stage ---------------------------------------------------------
FROM ubuntu:24.04
RUN apt-get update && apt-get upgrade -y && apt-get install -y --no-install-recommends \
    ca-certificates tzdata gpsbabel libimage-exiftool-perl curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -r -g 10001 tmpdrop \
    && useradd -r -u 10001 -g tmpdrop -d /nonexistent -s /sbin/nologin -c "" tmpdrop
WORKDIR /app
RUN mkdir -p /app/data && chown tmpdrop:tmpdrop /app/data
COPY --from=build /out/tmpdrop /usr/local/bin/tmpdrop
USER tmpdrop
ENV TMPDROP_ADDR=:8080 \
    TMPDROP_STORAGE_DIR=/app/data
VOLUME ["/app/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -sf http://127.0.0.1:8080/healthz > /dev/null || exit 1
ENTRYPOINT ["tmpdrop"]
