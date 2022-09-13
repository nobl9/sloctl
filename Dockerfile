FROM scratch
COPY ./sloctl /
ENTRYPOINT ["/sloctl"]