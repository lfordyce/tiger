# TIGER
TimescaleDB Benchmarking Analysis Tool

Menu
----

- [Setup](#setup)
- [Running](#running)
- [Configuration](#configuration)
- [Overview](#overview)
- [Tests](#tests)
- [Help](#help)

Setup
--------

### Docker

1. Make sure that you have installed **[Docker](https://docs.docker.com/install/)** and **[Docker Compose](https://docs.docker.com/compose/install/)**.
2. Clone the repository and execute the following commands:

```shell
# clone repository and navigate to the root of the project
git clone git@github.com:lfordyce/tiger.git && cd tiger

# start the TimescaleDB docker daemon with initial schema and dataset
make timescaledb

# build runtime docker image ( creates 'lfordyce/tiger' image )
make container
```

Running
--------

### Docker

> When using the `tiger` docker image, you can't just give the input csv data file name since the file will not be
available to the container as it runs. Instead, you must tell `tiger` to read `STDIN` by passing the file as `-`. Then
you pipe the actual file into the container with `<` or equivalent. This will cause the file to be redirected into the
container and be read by `tiger`.

Run the docker image created from the setup make target with input csv data, and network mode = `host`.
```shell
docker run --rm -i --network=host lfordyce/tiger run - <query_params.csv
```

Or alternatively, Docker Desktop 18.03+ for Windows and Mac supports `host.docker.internal` as a functioning alias for
localhost. Use this string inside the container to access the database with the `--host` flag.

```shell
docker run --rm -i lfordyce/tiger run - <query_params.csv --host host.docker.internal
```

Additionally, use the `-w` flag to specify the number of concurrent workers.

```shell
docker run --rm -i lfordyce/tiger run - <query_params.csv -w 5 --host host.docker.internal
```

### Docker in Win PowerShell

```shell
cat query_params.csv | docker run --rm -i --network=host lfordyce/tiger run -
```

### Go
> Minimum required version: 1.18

Passing in the filename

```shell
go run main.go run query_params.csv
```

Passing the filename in through STDIN

```shell
go run main.go run - < query_params.csv
```

Configuration
--------

- CLI optional parameters

|Option Name|Alias|Flag|Default|Description|
|-------------------------------|--|--------------------|----------------------|-------------------------------------------------|
|`workers`                      |-w|--workers           |`3`                   |Number of workers for concurrency work|
|`user`                         |  |--user              |`postgres`            |Postgres user (default "postgres")|
|`password`                     |  |--password          |`password`            |Postgres password (default "password")|
|`host`                         |  |--host              |`localhost`           |Postgres hostname (default "localhost")|
|`port`                         |  |--port              |`5432`                |Postgres port (default 5432)|
|`database`                     |  |--database          |`homework`            |Postgres database name (default "homework")|
|`CSV Host Header`              |  |--csv-host-hdr      |`hostname`            |The name of the CSV host id header (default "hostname")|
|`CSV Start Time Header`        |  |--csv-start-hdr     |`start_time`          |The name of the CSV start time header (default "start_time")|
|`CSV End Time Header`          |  |--csv-end-hdr       |`end_time`            |The name of the CSV end time header (default "end_time")|
|`CSV Timestamp format Header`  |  |--csv-ts-fmt        |`2006-01-02 15:04:05` |The go timestamp format of the CSV timestamp field (default "2006-01-02 15:04:05")|
|`log format`                   |  |--log-format        |                      |log output format|
|`log output`                   |  |--log-output        |`stderr`              |change the output for tiger logs, possible values are stderr,stdout,none,file[=./path.fileformat] (default "stderr")|
|`colored ouput`                |  |--no-color          |                      |disable colored output|
|`verbose`                      |-v|--verbose           |                      |enable verbose logging|

- Example usage of flag usage when provided with CSV header data different from defaults:
```shell
Hostname,start_time,end_time
host_000008,2017-01-01 08:59:22,2017-01-01 09:59:22
host_000001,2017-01-02 13:02:02,2017-01-02 14:02:02
...
```
- Run using the `--csv-host-hdr` flag:
```shell
docker run --rm -i --network=host lfordyce/tiger run - <bad_query_params.csv --csv-host-hdr Hostname
```

Overview
--------

### SQL query analysis

```postgresql
select time_bucket('1 minutes', ts) as one_minute,
       MAX(usage)                   as max_cpu,
       MIN(usage)                   as min_cpu
FROM cpu_usage
WHERE ts BETWEEN '2017-01-01 08:59:22' AND '2017-01-01 09:59:22'
  AND host = 'host_000008'
GROUP BY one_minute;
```

Tests
--------

### Run unit test
```shell
make test-short
```

### Run all tests: (unit, integration, bench). Requires TimescaleDB to be running
```shell
make tests
```


Help
--------

### CLI Help commands
* Root help
```shell
docker run --rm -i lfordyce/tiger --help
```
* Run cmd help
```shell
docker run --rm -i lfordyce/tiger run --help
```

Additional makefile help
```shell
make help
```

When running with network mode = host, the following should behave equivalently
```shell
docker run --rm -i --network=host lfordyce/tiger run - <query_params.csv -w 5 --host 0.0.0.0
```
* or
```shell
docker run --rm -i --network=host lfordyce/tiger run - <query_params.csv -w 5 --host 127.0.0.1
```