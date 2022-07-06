FROM timescale/timescaledb:latest-pg14

#RUN #mkdir -p /data
##
##copy all data to the container
#    COPY /data/* /data/
#    COPY proeve_schema.sql /docker-entrypoint-initdb.d/
#    RUN chown postgres /data
#    RUN chmod +w /data



RUN mkdir -p /data
#copy all data to the container
COPY /data/* /data/

COPY ./init-db.sql /docker-entrypoint-initdb.d/
RUN chown postgres /data
RUN chmod +w /data
#COPY data/cpu_usage.csv /var/lib/postgresql/csvs/