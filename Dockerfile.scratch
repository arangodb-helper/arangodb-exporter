FROM scratch
ARG GOARCH=amd64

COPY bin/linux/${GOARCH}/arangodb-exporter /app/

EXPOSE 9101

ENTRYPOINT ["/app/arangodb-exporter"]
