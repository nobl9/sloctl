FROM alpine:3.16.2
COPY ./sloctl /
ENTRYPOINT ["/sloctl"]