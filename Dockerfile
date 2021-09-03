FROM golang:1.16-alpine as builder

ENV CGO_ENABLED="0"

ADD . /app
WORKDIR /app
RUN go build -ldflags "-w -s" -o /juicyrout

FROM alpine:3.14

COPY --from=builder /juicyrout /juicyrout
COPY ./phishlets /phishlets

ENTRYPOINT ["/juicyrout"]