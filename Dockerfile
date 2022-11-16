FROM --platform=linux/x86_64 alpine:latest
ENV GLIBC_REPO=https://github.com/sgerrand/alpine-pkg-glibc
ENV GLIBC_VERSION=2.35-r0
RUN set -ex && \
    apk --update add libstdc++ curl ca-certificates && \
    for pkg in glibc-${GLIBC_VERSION} glibc-bin-${GLIBC_VERSION}; \
        do curl -sSL ${GLIBC_REPO}/releases/download/${GLIBC_VERSION}/${pkg}.apk -o /tmp/${pkg}.apk; done && \
    apk add --force-overwrite --allow-untrusted /tmp/*.apk && \
    rm -v /tmp/*.apk && \
    /usr/glibc-compat/sbin/ldconfig /lib /usr/glibc-compat/lib
COPY ./sloctl /usr/local/bin/sloctl
RUN adduser -D appuser
RUN mkdir -p /home/appuser/.config/nobl9
RUN chown -R appuser:appuser /home/appuser/.config/nobl9
USER appuser
ENTRYPOINT ["sloctl"]