FROM golang:alpine as builder
RUN apk update && apk add --no-cache --update git build-base

RUN git clone https://git.zx2c4.com/wireguard-go && \
    cd wireguard-go && \
    make && \
    make install

WORKDIR /app
COPY . ./
ARG AGENT_VERSION
RUN make

FROM alpine

RUN apk update && apk add --no-cache --update iptables wireguard-tools
COPY --from=builder /usr/bin/wireguard-go /usr/bin/wg* /usr/bin/
COPY --from=builder ./app/syntropy_agent /syntropy_agent

ENTRYPOINT [ "./syntropy_agent"]
