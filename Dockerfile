FROM golang:1.26.4-alpine AS builder

ENV CGO_ENABLED=1

RUN mkdir /mailman
RUN mkdir /mailman/db
RUN mkdir /mailman/log
RUN mkdir /mailman/migrations

WORKDIR /mailman

COPY ./mailman /mailman/mailman
COPY ./vendor /mailman/vendor
COPY ./go.mod /mailman
COPY ./go.sum /mailman
  
# COPY ./certs/ /go-webapp/certs/
# COPY ./certs/go-dev.nl.pem /usr/local/share/ca-certificates/go-dev.nl.pem
# RUN cat /go-webapp/certs/root_go-dev.nl.crt >> /etc/ssl/certs/ca-certificates.crt
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
RUN apk --update add --no-cache tzdata openssl
ENV TZ=Europe/Amsterdam
# COPY ./certs/keychain.pem /usr/local/share/ca-certificates/go-dev.nl.crt
# RUN cat /usr/local/share/ca-certificates/go-dev.nl.crt >> /etc/ssl/certs/ca-certificates.crt # && apk --no-cache add curl

COPY ./certs/go-dev.crt              /usr/local/share/ca-certificates/go-dev.nl.crt
COPY ./certs/intermediate.go-dev.crt /usr/local/share/ca-certificates/intermediate.go-dev.nl.crt
COPY ./certs/root.go-dev.crt         /usr/local/share/ca-certificates/root.go-dev.crt
COPY ./certs/go-dev.key              /usr/local/share/ca-certificates/go-dev.nl.key

RUN update-ca-certificates

RUN apk add --no-cache \
    # Important: required for go-sqlite3
    gcc \
    # Required for Alpine
    musl-dev

RUN export GO111MODULE=on
RUN go build -o /mailman/server -ldflags='-s -w -extldflags "-static"' --tags="docker" /mailman/mailman

FROM scratch
LABEL maintainer="Nigel"
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Europe/Amsterdam

# COPY ./ssl/ /etc/ssl/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /mailman/server /mailman/server

WORKDIR /mailman

ENV MAILMAN_INTERFACE=0.0.0.0
ENV MAILMAN_PORT=8080

EXPOSE 8080

VOLUME [ "/mailman/db" ]
VOLUME [ "/mailman/log" ]

CMD [ "/mailman/server" ]