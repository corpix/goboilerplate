FROM golang:1.15 as builder

WORKDIR /go/src/git.backbone/corpix/goboilerplate
COPY    . .
ENV     CGO_ENABLED 0
RUN     make build

FROM alpine:latest

RUN  mkdir          /etc/goboilerplate
COPY --from=builder /go/src/git.backbone/corpix/goboilerplate/config.yaml /etc/goboilerplate/config.yaml
COPY --from=builder /go/src/git.backbone/corpix/goboilerplate/main        /usr/bin/goboilerplate

#EXPOSE ?/tcp

CMD [                                \
    "/usr/bin/goboilerplate",        \
    "--config",                      \
    "/etc/goboilerplate/config.yaml" \
]
