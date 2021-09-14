FROM golang:alpine as builder
RUN apk update && apk add --no-cache --update git build-base
WORKDIR /app
COPY . ./
ARG AGENT_VERSION
ENV AGENT_VERSION=$AGENT_VERSION
RUN make

FROM alpine

RUN apk update && apk add --no-cache --update iptables
COPY --from=builder ./app/sag /sag

ENTRYPOINT [ "./sag"]
