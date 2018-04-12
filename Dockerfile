FROM scratch

COPY bin/linux/amd64/arangodb_exporter /app/

EXPOSE 9101

ENTRYPOINT ["/app/arangodb_exporter"]
