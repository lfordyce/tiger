# TIGER

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

Startup TimescaleDB with initial schema and dataset in docker

```shell
make timescaledb
```

Build the runtime docker image

```shell
make container
```

Running
--------

### Docker

When using the `tiger` docker image, you can't just give the input csv data file name since the file will not be
available to the container as it runs. Instead you must tell `tiger` to read `stdin` by passing the file as `-`. Then
you pipe the actual file into the container with `<` or equivalent. This will cause the file to be redirected into the
container and be read by `tiger`.
Running with network mode = host.

```shell
docker run --rm -i --network=host lfordyce/tiger run - <query_params.csv
```

Or alternatively, Docker Desktop 18.03+ for Windows and Mac supports `host.docker.internal` as a functioning alias for
localhost. Use this string inside your containers to access your host machine.

```shell
docker run --rm -i lfordyce/tiger run - <query_params.csv --host host.docker.internal
```

Additional use the `-w` flag to specify the number of concurrent workers

```shell
docker run --rm -i lfordyce/tiger run - <query_params.csv -w 5 --host host.docker.internal
```

### Docker in Win PowerShell

```shell
cat query_params.csv | docker run --rm -i --network=host lfordyce/tiger run -
```

### GO

Passing in the filename

```shell
go run main.go run query_params.csv
```

Passing the filename in through stdin

```shell
go run main.go run - < query_params.csv
```

Configuration
--------

- CLI optional parameters

|Option Name|Alias|Flag|Default|Description|
|-------------------------------|--|-------------------|----------------|-------------------------------------------------|
|`workers`                      |-w|--workers           |`3`            |Number of workers for concurrency work|
|`user`                         |  |--user              |`postgres`     |Postgres user (default "postgres")|
|`password`                     |  |--password          |`password`     |Postgres password (default "password")|
|`host`                         |  |--host              |`localhost`    |Postgres hostname (default "localhost")|
|`port`                         |  |--port              |`5432`         |Postgres port (default 5432)|
|`database`                     |  |--database          |`homework`     |Postgres database name (default "homework")|
|`CSV Host Header`              |  |--csv-host-hdr      |`hostname`     |The name of the CSV host id header (default "hostname")|
|`CSV Start Time Header`        |  |--csv-start-hdr     |`start_time`   |The name of the CSV start time header (default "start_time")|
|`CSV End Time Header`          |  |--csv-end-hdr       |`end_time`     |The name of the CSV end time header (default "end_time")|
|`CSV Timestamp format Header`  |  |--csv-ts-fmt        |`2006-01-02 15:04:05` |The go timestamp format of the CSV timestamp field (default "2006-01-02 15:04:05")|
|`log format`                   |  |--log-format        |                |log output format|
|`log output`                   |  |--log-output        |`stderr`        |change the output for tiger logs, possible values are stderr,stdout,none,file[=./path.fileformat] (default "stderr")|
|`colored ouput`                |  |--no-color          |                |disable colored output|
|`verbose`                      |-v|--verbose           |                |enable verbose logging|

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