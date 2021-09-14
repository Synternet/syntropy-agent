FROM golang:alpine as builder
RUN apk update && apk add --no-cache --update git build-base
WORKDIR /app
COPY . ./
ARG GIT_REF
RUN echo $GIT_REF
RUN make

FROM alpine

RUN apk update && apk add --no-cache --update iptables
COPY --from=builder ./app/sag /sag

ENTRYPOINT [ "./sag"]
