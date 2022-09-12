FROM scratch
WORKDIR .
COPY sloctl/sloctl /
RUN "install -o root -g root -m 0755 /sloctl /usr/local/bin/sloctl"
ENTRYPOINT ["/sloctl"]