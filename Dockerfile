FROM golang:1.16 AS build

ADD . /opt/app

WORKDIR /opt/app

RUN go build ./main.go


FROM ubuntu:20.04

RUN apt-get -y update && apt-get install -y tzdata

ENV PGVER 12

COPY schema.sql /

RUN apt-get -y update && apt-get install -y postgresql-$PGVER

USER postgres

RUN /etc/init.d/postgresql start &&\
    psql -U postgres -d postgres -a -f schema.sql &&\
    /etc/init.d/postgresql stop

RUN echo "host all  all    0.0.0.0/0  md5" >> /etc/postgresql/$PGVER/main/pg_hba.conf

RUN echo "listen_addresses='*'\nsynchronous_commit = off\nfsync = off\nshared_buffers = 256MB\neffective_cache_size = 1536MB\n" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "wal_buffers = 1MB\nwal_writer_delay = 50ms\nrandom_page_cost = 1.0\nmax_connections = 100\nwork_mem = 8MB\nmaintenance_work_mem = 128MB\ncpu_tuple_cost = 0.0030\ncpu_index_tuple_cost = 0.0010\ncpu_operator_cost = 0.0005" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "full_page_writes = off" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "log_statement = none" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "log_duration = off " >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "log_lock_waits = on" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "log_min_duration_statement = 500000" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "log_filename = 'query.log'" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "log_directory = '/var/log/postgresql'" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "log_destination = 'csvlog'" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "logging_collector = on" >> /etc/postgresql/$PGVER/main/postgresql.conf
RUN echo "log_temp_files = '-1'" >> /etc/postgresql/$PGVER/main/postgresql.conf

EXPOSE 5432

VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

USER root

COPY --from=build /opt/app/main /bin/

EXPOSE 5000

CMD service postgresql start && main
