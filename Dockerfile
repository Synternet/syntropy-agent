FROM golang:alpine as builder
RUN apk update && apk add --no-cache --update git build-base
WORKDIR /app
COPY . ./
ARG AGENT_VERSION
RUN make

FROM alpine

RUN apk update && apk add --no-cache --update iptables
COPY --from=builder ./app/syntropy_agent /syntropy_agent

ENTRYPOINT [ "./sag"]
