* Test Query:
```postgresql
select time_bucket('1 minutes', ts) as one_minute, 
       MAX(usage) as max_cpu, 
       MIN(usage) as min_cpu 
FROM cpu_usage 
WHERE ts BETWEEN '2017-01-01 08:59:22' AND '2017-01-01 09:59:22' 
  AND host = 'host_000008' 
GROUP BY one_minute;
```

* Run:
* From stdin
```shell
go run . < query_params.csv
```
* from arg
```shell
go run . query_params.csv
```
