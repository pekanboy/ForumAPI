FROM golang:1.16 AS build

ADD . /opt/app

WORKDIR /opt/app

RUN go build ./main.go


FROM ubuntu:20.04

RUN apt-get -y update && apt-get install -y tzdata

ENV PGVER 12

RUN apt-get -y update && apt-get install -y postgresql-$PGVER

USER postgres

RUN /etc/init.d/postgresql start &&\
    psql --command "CREATE USER docker WITH SUPERUSER PASSWORD 'docker';" &&\
    createdb -O docker docker &&\
    /etc/init.d/postgresql stop

RUN echo "host all  all    0.0.0.0/0  md5" >> /etc/postgresql/$PGVER/main/pg_hba.conf

RUN echo "listen_addresses='*'\nsynchronous_commit = off\nfsync = off\nmax_connections = 100\nshared_buffers = 1GB\neffective_cache_size = 3GB\nmaintenance_work_mem = 256MB\ncheckpoint_completion_target = 0.7\nwal_buffers = 16MB\ndefault_statistics_target = 100\nrandom_page_cost = 1.1\neffective_io_concurrency = 200\nwork_mem = 10485kB\nmin_wal_size = 1GB\nmax_wal_size = 4GB\nmax_worker_processes = 2\nmax_parallel_workers_per_gather = 1\nmax_parallel_workers = 2\nmax_parallel_maintenance_workers = 1" >> /etc/postgresql/$PGVER/main/postgresql.conf

EXPOSE 5432

VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

USER root

WORKDIR /usr/src/app

COPY . .

COPY --from=build /opt/app/main .

EXPOSE 5000

CMD service postgresql start && psql -h localhost -d docker -U docker -p 5432 -a -q -f ./schema.sql && ./main
