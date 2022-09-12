FROM scratch
COPY /usr/local/bin/sloctl /
ENTRYPOINT ["/sloctl"]