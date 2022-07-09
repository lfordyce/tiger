FROM timescale/timescaledb:latest-pg14

RUN mkdir -p /data
#copy all data to the container
COPY /data/* /data/

COPY ./init-db.sql /docker-entrypoint-initdb.d/
RUN chown postgres /data
RUN chmod +w /data