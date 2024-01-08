FROM golang:1.11-alpine as builder

WORKDIR /go/src/github.com/lomik/prometheus-png

RUN apk --no-cache add make pkgconfig cairo-dev gcc g++

COPY . .
RUN make

FROM ubuntu:18.04 as fonts
RUN apt update && apt install -y fonts-roboto

FROM alpine:latest

COPY --from=fonts /usr/share/fonts/truetype/roboto/hinted /usr/share/fonts/ttf-roboto-hinted

RUN apk --no-cache add ca-certificates cairo fontconfig ttf-dejavu ttf-freefont && \
    fc-cache -f

WORKDIR /

EXPOSE 8080/tcp

COPY --from=builder /go/src/github.com/lomik/prometheus-png/prometheus-png /usr/bin/prometheus-png

ENTRYPOINT ["prometheus-png"]
