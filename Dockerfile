FROM golang:1.24-alpine as builder

RUN apk update
RUN apk upgrade
RUN apk add --no-cache git make

RUN git --version

WORKDIR /work

COPY go.mod .
COPY . .
RUN go mod download
RUN go build

FROM alpine:latest
LABEL maintainer="Thomas von Dein <git@daemon.de>"

WORKDIR /app
COPY --from=builder /work/io-exporter /app/io-exporter

USER 1001:1001

ENTRYPOINT ["/app/io-exporter"]
CMD ["-h"]
