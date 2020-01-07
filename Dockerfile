FROM golang:1.13-alpine as builder

RUN apk update && apk add --no-cache sqlite-dev make gcc libc-dev tzdata git bash
RUN adduser -D -g '' letarette

WORKDIR /go/src/app
COPY . .
RUN ./build.sh
RUN go run example/loader.go recipies.db example/pg24384.json

FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /bin/sh /bin/sh
COPY --from=builder /lib/ld-musl-x86_64.so.1 /lib/ld-musl-x86_64.so.1

COPY --from=builder /go/src/app/recipies.db /example/
COPY example/*.sql /example/
COPY --from=builder /go/src/app/lrsql /

USER letarette
ENV LRSQL_NATS_URLS=nats://natsserver:4222

CMD [ "/lrsql" ]
