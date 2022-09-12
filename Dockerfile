FROM scratch
WORKDIR .
COPY sloctl /
RUN "install -o root -g root -m 0755 sloctl/sloctl /usr/local/bin/sloctl"
ENTRYPOINT ["/sloctl"]