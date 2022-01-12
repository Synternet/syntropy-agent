FROM alpine

# Allow adding additional packages without modifying Dockefile
# e.g. # docker build --build-arg packages="vim bird" ./
ARG packages
RUN apk update && apk add --no-cache --update iptables wireguard-tools $packages

# Prepare binaries for all targets
RUN mkdir /tmp/target
COPY ./target /tmp/target

# Copy only required target architecture
RUN  apkArch="$(apk --print-arch)"; \
     case "$apkArch" in \
            x86_64) export ARCH='x86_64' ;; \
            aarch64) export ARCH='arm64' ;; \
            *) export ARCH='unsupported' ;; \
        esac; \
        cp /tmp/target/$ARCH/* /usr/bin

# Cleanup
RUN rm -rf /tmp/target

ENTRYPOINT [ "/usr/bin/syntropy_agent"]
