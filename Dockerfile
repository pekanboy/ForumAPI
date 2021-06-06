FROM golang:1.16 AS build

ADD . /opt/app

WORKDIR /opt/app

sudo go build ./main.go


FROM ubuntu:20.04

sudo apt-get -y update && apt-get install -y tzdata

ENV PGVER 12

COPY schema.sql /

sudo apt-get -y update && apt-get install -y postgresql-12

USER postgres

sudo sudo /etc/init.d/postgresql start &&\
    psql -U postgres -d postgres -a -f schema.sql &&\
    sudo /etc/init.d/postgresql stop

sudo echo "host all  all    0.0.0.0/0  md5" >> sudo /etc/postgresql/12/main/pg_hba.conf

sudo echo "listen_addresses='*'\nsynchronous_commit = off\nfsync = off\nshared_buffers = 256MB\neffective_cache_size = 1536MB\n" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "wal_buffers = 1MB\nwal_writer_delay = 50ms\nrandom_page_cost = 1.0\nmax_connections = 100\nwork_mem = 8MB\nmaintenance_work_mem = 128MB\ncpu_tuple_cost = 0.0030\ncpu_index_tuple_cost = 0.0010\ncpu_operator_cost = 0.0005" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "full_page_writes = off" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "log_statement = none" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "log_duration = on " >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "log_lock_waits = on" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "log_min_duration_statement = 500" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "log_filename = 'query.log'" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "log_directory = '/var/log/postgresql'" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "log_destination = 'csvlog'" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "logging_collector = on" >> sudo /etc/postgresql/12/main/postgresql.conf
sudo echo "log_temp_files = '-1'" >> sudo /etc/postgresql/12/main/postgresql.conf

EXPOSE 5432

VOLUME  ["sudo /etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

USER root

COPY --from=build /opt/app/main /bin/

EXPOSE 5000

CMD service postgresql start && main
