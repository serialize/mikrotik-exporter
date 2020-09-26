FROM frolvlad/alpine-glibc

EXPOSE 9436

COPY scripts/start.sh /app/
COPY mikrotik-exporter /app/mikrotik-exporter

RUN chmod 755 /app/*

ENTRYPOINT ["/app/start.sh"]