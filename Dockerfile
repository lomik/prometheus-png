FROM golang:alpine as builder

WORKDIR /go/src/github.com/lomik/prometheus-png

COPY . .

RUN apk --no-cache add make pkgconfig cairo-dev gcc g++

RUN make

FROM alpine:latest

RUN apk --no-cache add ca-certificates cairo fontconfig ttf-dejavu
WORKDIR /

EXPOSE 8080/tcp

COPY --from=builder /go/src/github.com/lomik/prometheus-png/prometheus-png /usr/bin/prometheus-png

ENTRYPOINT ["prometheus-png"]
