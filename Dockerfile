FROM curlimages/curl:latest AS builder
ARG VERSION
RUN curl -sL https://github.com/nobl9/sloctl/releases/download/v$VERSION/sloctl-linux-$VERSION -o /tmp/sloctl
RUN chmod +x /tmp/sloctl

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /tmp/sloctl /usr/bin/
ENTRYPOINT ["sloctl"]