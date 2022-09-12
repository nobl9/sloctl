FROM scratch
WORKDIR .
COPY sloctl /
ENTRYPOINT ["/sloctl/sloctl"]