CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE TABLE cpu_usage
(
    ts    TIMESTAMPTZ,
    host  TEXT,
    usage DOUBLE PRECISION
);
SELECT create_hypertable('cpu_usage', 'ts');

COPY cpu_usage FROM '/var/lib/postgresql/csvs/cpu_usage.csv' DELIMITER ',' CSV HEADER;

CREATE OR REPLACE FUNCTION bench(hostname_id TEXT, ts_start TIMESTAMPTZ, ts_end TIMESTAMPTZ)
    RETURNS TABLE
            (
                elapsed DOUBLE PRECISION
            )
    LANGUAGE PLPGSQL
AS
$$
DECLARE
    _timing   TIMESTAMPTZ;
    _start_ts TIMESTAMPTZ;
    _end_ts   TIMESTAMPTZ;
    _overhead NUMERIC; -- in ms
    _delta    DOUBLE PRECISION;
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS _bench_results
    (
        elapsed DOUBLE PRECISION
    );

    _timing := clock_timestamp();
    _start_ts := clock_timestamp();
    _end_ts := clock_timestamp();
    -- take minimum duration as conservative estimate
    _overhead := 1000 * extract(epoch FROM LEAST(_start_ts - _timing
        , _end_ts - _start_ts));

    _start_ts := clock_timestamp();

    -- lookup query: max cpu usage and min cpu usage of the given hostname for every minute int the range specified.
    PERFORM time_bucket('1 minutes', ts) as one_minute,
        MAX(usage) as max_cpu,
        MIN(usage) as min_cpu
    FROM cpu_usage
    WHERE ts BETWEEN ts_start AND ts_end
      AND host = hostname_id
    GROUP BY one_minute;

    _end_ts := clock_timestamp();
    _delta = 1000 * (extract(epoch FROM _end_ts - _start_ts)) - _overhead;
    RAISE NOTICE 'Timing overhead in ms = %', _overhead;
    RAISE NOTICE 'Execution time in ms = %' , _delta;
    INSERT INTO _bench_results VALUES (_delta);

    RETURN QUERY SELECT _bench_results.elapsed FROM _bench_results;
    DROP TABLE IF EXISTS _bench_results;
END;
$$;