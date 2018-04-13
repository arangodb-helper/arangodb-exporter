FROM scratch

COPY bin/linux/amd64/arangodb-exporter /app/

EXPOSE 9101

ENTRYPOINT ["/app/arangodb-exporter"]
