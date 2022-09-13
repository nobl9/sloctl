FROM --platform=linux/amd64 scratch
COPY ./sloctl /
ENTRYPOINT ["/sloctl"]