FROM golang:1.26.4-alpine AS builder

ENV CGO_ENABLED=1

RUN mkdir /mailman
RUN mkdir /mailman/db
RUN mkdir /mailman/log
RUN mkdir /mailman/migrations

WORKDIR /mailman

COPY ./mailman /mailman/mailman
COPY ./templates /mailman/templates
COPY ./go.mod /mailman
COPY ./go.sum /mailman
  
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN apk --update add --no-cache tzdata openssl
ENV TZ=Europe/Amsterdam

RUN update-ca-certificates

RUN apk add --no-cache \
    # Important: required for go-sqlite3
    gcc \
    # Required for Alpine
    musl-dev

RUN export GO111MODULE=on

# build static binary to avoid linker issues
RUN go build -o /mailman/server -ldflags='-s -w -extldflags "-static"' --tags="docker" /mailman/mailman

FROM scratch
LABEL maintainer="Nigel"
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Europe/Amsterdam

# COPY ssl certs to /etc/ssl/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /mailman/server /mailman/server

WORKDIR /mailman

ENV DOCKER=1
ENV MAILMAN_INTERFACE=0.0.0.0
ENV MAILMAN_PORT=8080
ENV MAILMAN_SIEVE_TEMPLATE=/mailman/templates/tmp/docker-mailserver/before.dovecot.sieve.tmpl

EXPOSE 8080

VOLUME [ "/mailman/db" ]
VOLUME [ "/mailman/log" ]

CMD [ "/mailman/server" ]