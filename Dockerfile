# Builder
FROM golang:1.18.3-alpine3.16 as builder

RUN apk update && apk upgrade && \
    apk --update add git make

WORKDIR /home

COPY . .

# Build executables called arbot
RUN make build

# Distribution
FROM alpine:latest

RUN apk update && apk upgrade && \
    apk --update --no-cache add tzdata && \
    mkdir -p /home

WORKDIR /home

EXPOSE 8080

COPY --from=builder /home/config.yml .
COPY --from=builder /home/arbot .

CMD ["./arbot"]