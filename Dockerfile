FROM alpine
RUN apk add gcompat
COPY ./sloctl /usr/local/bin/sloctl
RUN adduser -S appuser
USER appuser
ENTRYPOINT ["sloctl"]