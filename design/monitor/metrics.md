# Contents
- [Abstract](#abstract)
- [Functional Design](#functional-design)
  - [General](#general)
  - [API Functional Design](#api-functional-design)
- [Technical Design](#technical-design)
  - [API](#api)
    - [System Resources](#system-resources)
    - [System Performance](#system-performance)
    - [Dashboard](#dashboard)
    - [App Performance](#app-performance)
  - [Metrics](#metrics)
    - [Writing metrics](#writing-metrics)
    - [List of Metrics](#list-of-metrics)
    - [Metrics View](#metrics-view)

# Abstract
As a system architect I want to know which metrics are needed for the monitor app, and the API to query them, so that [user requirements for CE monitoring](https://github.com/heeus/inv-monitoring/tree/master/20220503-user-reqs) can be implemented

# Functional Design
## General
- Monitor App Frontend requests metrics from Backend using API
- Monitor App performs required calculations over metrics if needed (rate, diff, etc and shows charts/summaries etc

## API Functional Design
- General
  - select list of nodes (vvms and dbs)
- Time-series charts
  - select list of metrics by the time range (from..till)
- Dashboard current values / gauges
  - select last metric value (select top 1 metric from the time range order by time desc)
- Dashboard current values / rates (CPU load, IOPS)
  - select first and last metric value from the interval
   Dashboard: Applications Overview
  - select the list of apps with their versions, partitions, uptime, avg RPS
- Sys Performance IOPS: Worst apps
  - Top 5 by request time (Get, GetBatch, Read, Put, PutBatch)
  - Top 5 by RPS (Get, GetBatch, Read, Put, PutBatch)
  - Bottom 5 bycache hits (Get, GetBatch)
  - Top 5 by batch size (PutBatch)
- App: top 10 slow projectors
  - select top 10 projector partitions (Name + Partition + Lag)
- App: Partitions balance
  - select number of queries and commands for every app partition for the specified period

# Technical Design

## API
The following query functions are available in the API:
- `q.monitor.GetNodes` ([{nodename: 'worker1', vvm: true},...])
- `q.monitor.GetApps` ([{app: 'sys/monitor, version: '1.2.3', partitions: 1, uptime: 123123123}])
- `q.monitor.GetMetrics` (return all values for requested metrics over time interval for given app)
  - in:
    - from
    - till
    - app
    - list of metrics
    - list of vvms
  - out: array of objects:
    - metric_name
    - app
    - month
    - timestamp
    - value

- `q.monitor.GetPartitionsBalance` (return the data to show the Partition Balance over time interval for given app, see below)
- `q.monitor.GetWorstApps` (return the "IOPS/Worst Apps" data over time interval, see below)


### System Resources

CPU Usage
- Gets the list of ['node_cpu_idle_seconds_total'] metrics from `q.monitor.GetMetrics` for app 'sys', given nodes and interval
- Calculates `rate` over values

Mem Usage
- Gets the list of ['node_memory_memavailable_bytes', 'node_memory_memtotal_bytes'] metrics from `q.monitor.GetMetrics` for app 'sys', given nodes and interval
- Calculates percentage over values (avail/total)

Disk Usage
- Gets the list of ['node_filesystem_free_bytes', 'node_filesystem_size_bytes'] metrics from `q.monitor.GetMetrics` for app 'sys', given nodes and interval
- Calculates percentage over values (avail/total)

Disk I/O
- Gets the list of ['node_disk_read_bytes_total', 'node_disk_write_bytes_total'] metrics from `q.monitor.GetMetrics` for app 'sys', given nodes and interval
- Calculates the rate of read+write ops to show datasize per second

Disk IOPS
- Gets the list of ['node_disk_reads_completed_total', 'node_disk_writes_completed_total'] metrics from `q.monitor.GetMetrics` for app 'sys', given nodes and interval
- Calculates the rate of read+write ops to show number of ops per second


### System Performance

RPS
- Gets the list of ['heeus_cp_commands_total', 'heeus_qp_commands_total'] metrics from `q.monitor.GetMetrics` for app 'sys' and given interval
- Calculates `rate` over values to show number per second

Status Codes
- Gets the list of ['heeus_http_status_2xx_total', 'heeus_http_status_4xx_total', 'heeus_http_status_5xx_total', 'heeus_http_status_503_total'] metrics from `q.monitor.GetMetrics` for app 'sys' and given interval
- Calculates `diff` over values to show number per interval

IOPS
- Gets the list of ['heeus_istoragecache_get_total', 'heeus_istoragecache_getbatch_total', 'heeus_istoragecache_put_total',  'heeus_istoragecache_putbatch_total', 'heeus_istoragecache_read_total'] metrics from `q.monitor.GetMetrics` for app 'sys' and given interval
- Calculates `rate` over values to show number per second

IOPS: cache hits
- Gets the list of 'heeus_istoragecache_get_total', 'heeus_istoragecache_get_cached_total', 'heeus_istoragecache_getbatch_total', 'heeus_istoragecache_getbatch_cached_total'] metrics from `q.monitor.GetMetrics` for app 'sys' and given interval
- Calculates percentage over `diff` of values (get_cached/get; getbatch_cached/getbatch)

Worst Apps
- Gets the report from `q.monitor.GetWorstApps` fuction (interval)
- Internally the function works with the list of apps and metrics from the previous two paragraphs

### Dashboard

System Resources Overview
- CPU
  - Same as [CPU Usage](#cpu-usage), but only rate between first and last value
- Memory
  - Same as [Memory Usage](#mem-usage), but only read last value
- Disk
  - Same as [Disk Usage](#disk-usage), but only read last value
- IOPS
  - Same as [IOPS](#iops), but only rate between first and last value

System Performance Overview
- RPS
  - Same as [RPS](#rps) but only rate between first and last value
- Status Codes
  - Same as [Status Codes](#status-codes) but only diff between first and last value
- IOPS
  - Same as [IOPS](#iops) but only rate between first and last value

Applications Overview
- Gets the list of apps with `q.monitor.GetApps` function
- RPS for every app is got in the same way with [App Rps](#app-rps), but rate between first and last value


### App Performance

App RPS
- Gets the list of ['heeus_cp_commands_total', 'heeus_qp_commands_total'] metrics from `q.monitor.GetMetrics` for given app and interval
- Calculates `rate` over values to show number per second

App Status Codes
- Gets the list of ['heeus_http_status_2xx_total', 'heeus_http_status_4xx_total', 'heeus_http_status_5xx_total', 'heeus_http_status_503_total'] metrics from `q.monitor.GetMetrics` for given app and interval
- Calculates `diff` over values to show number per interval

App Status Codes / Command Processor
- The same, but different metrics ['heeus_cp_http_status_2xx_total', 'heeus_cp_http_status_4xx_total', 'heeus_cp_http_status_5xx_total', 'heeus_cp_http_status_503_total']

App Status Codes / Query Processor
- The same, but different metrics ['heeus_qp_http_status_2xx_total', 'heeus_qp_http_status_4xx_total', 'heeus_qp_http_status_5xx_total', 'heeus_qp_http_status_503_total']

App Execution Time / Command Processor
- Gets the list of ['heeus_cp_commands_total', 'heeus_cp_commands_seconds', 'heeus_cp_exec_seconds', 'heeus_cp_validate_seconds', 'heeus_cp_putplog_seconds'] metrics from `q.monitor.GetMetrics` for given app and interval
- Calculates `diff` over values to show the execution time: diff(seconds)/diff(total)

App Execution Time / Query Processor
- Gets the list of ['heeus_qp_queries_total',
                'heeus_qp_queries_seconds', 'heeus_qp_build_seconds',
                'heeus_qp_exec_seconds', 'heeus_qp_exec_fields_seconds',
                'heeus_qp_exec_enrich_seconds', 'heeus_qp_exec_filter_seconds',
                'heeus_qp_exec_order_seconds','heeus_qp_exec_count_seconds',
                'heeus_qp_exec_send_seconds'] metrics from `q.monitor.GetMetrics` for given app and interval
- Calculates `diff` over values to show the execution time: diff(seconds)/diff(total)

App Partitions Balance
- separate query function `q.monitor.GetPartitionsBalance(interval)` which interally selects difference between partition metrics over the range
- in:
  - range
  - appName
- out:
  [
    {partition: 'P1', queries: 100, commands: 20},
    ...
  ]
- metrics
  - ['heeus_partition_cp_commands_total', 'heeus_partition_qp_commands_total']
  - Note that for this case we should add `partition` to the metric, e.g. metrics *may* have partition

App Top 10 Slow Projectors
???

App Storage / IOPS
- Gets the list of ['heeus_istoragecache_get_total',
                'heeus_istoragecache_getbatch_total', 'heeus_istoragecache_put_total',
                'heeus_istoragecache_putbatch_total', 'heeus_istoragecache_read_total'] metrics from `q.monitor.GetMetrics` for given app and interval
- Calculates `rate` over values to show the ops per seconds

App Storage / Execution Time
- Gets the list of ['heeus_istoragecache_get_seconds', 'heeus_istoragecache_get_total',
                        'heeus_istoragecache_getbatch_seconds', 'heeus_istoragecache_getbatch_total',
                        'heeus_istoragecache_put_seconds', 'heeus_istoragecache_put_total',
                        'heeus_istoragecache_putbatch_seconds', 'heeus_istoragecache_putbatch_total',
                        'heeus_istoragecache_read_seconds', 'heeus_istoragecache_read_total'] metrics from `q.monitor.GetMetrics` for given app and interval
- Calculates `diff` over values to show the execution time: diff(seconds)/diff(total)


App Storage / Cache hits
- Gets the list of ['heeus_istoragecache_get_total', 'heeus_istoragecache_get_cached_total',
                'heeus_istoragecache_getbatch_total', 'heeus_istoragecache_getbatch_cached_total'] metrics from `q.monitor.GetMetrics` for given app and interval
- Calculates `diff` over values to show the execution time: diff(cached)/diff(total)

## Metrics
### Writing Metrics
Metrics are periodically scraped by Monitor app and saved in `monitor.MetricsView` with the timestamps

### List of Metrics
|                      Metric                       |  VVM  | Partitioned |
| ------------------------------------------------- | ----- | ----------- |
| heeus_http_status_2xx_total                       | yes   | no
| heeus_http_status_4xx_total                       | yes   | no
| heeus_http_status_5xx_total                       | yes   | no
| heeus_http_status_503_total                       | yes   | no
| heeus_cp_http_status_503_total                    | yes   | no
| heeus_cp_http_status_4xx_total                    | yes   | no
| heeus_cp_http_status_5xx_total                    | yes   | no
| heeus_cp_http_status_503_total                    | yes   | no
| heeus_qp_http_status_503_total                    | yes   | no
| heeus_qp_http_status_4xx_total                    | yes   | no
| heeus_qp_http_status_5xx_total                    | yes   | no
| heeus_qp_http_status_503_total                    | yes   | no
| heeus_istoragecache_get_total                     | yes   | no
| heeus_istoragecache_get_cached_total              | yes   | no
| heeus_istoragecache_getbatch_total                | yes   | no
| heeus_istoragecache_getbatch_cached_total         | yes   | no
| heeus_istoragecache_put_total                     | yes   | no
| heeus_istoragecache_putbatch_total                | yes   | no
| heeus_istoragecache_read_total                    | yes   | no
| heeus_istoragecache_get_seconds                   | yes   | no
| heeus_istoragecache_getbatch_seconds              | yes   | no
| heeus_istoragecache_put_seconds                   | yes   | no
| heeus_istoragecache_putbatch_seconds              | yes   | no
| heeus_istoragecache_read_seconds                  | yes   | no
| heeus_cp_commands_total                           | yes   | no
| heeus_cp_commands_seconds                         | yes   | no
| heeus_cp_exec_seconds                             | yes   | no
| heeus_cp_validate_seconds                         | yes   | no
| heeus_cp_putplog_seconds                          | yes   | no
| heeus_qp_queries_total                            | yes   | no
| heeus_qp_queries_seconds                          | yes   | no
| heeus_qp_build_seconds                            | yes   | no
| heeus_qp_exec_seconds                             | yes   | no
| heeus_qp_exec_fields_seconds                      | yes   | no
| heeus_qp_exec_enrich_seconds                      | yes   | no
| heeus_qp_exec_filter_seconds                      | yes   | no
| heeus_qp_exec_order_seconds                       | yes   | no
| heeus_qp_exec_count_seconds                       | yes   | no
| heeus_qp_exec_send_seconds                        | yes   | no
| heeus_partition_cp_commands_total                 | yes   | yes
| heeus_partition_qp_commands_total                 | yes   | yes
| node_cpu_idle_seconds_total                       | no    | no
| node_memory_memavailable_bytes                    | no    | no
| node_memory_memtotal_bytes                        | no    | no
| node_filesystem_free_bytes                        | no    | no
| node_filesystem_size_bytes                        | no    | no
| node_disk_read_bytes_total                        | no    | no
| node_disk_write_bytes_total                       | no    | no
| node_disk_reads_completed_total                   | no    | no
| node_disk_writes_completed_total                  | no    | no

### Metrics View
- PK: app, day_in_month
- CC: metric_name, timestamp, node
- partition
- value: float64

Partition size calculation:
- Scrape every 15 seconds = 5760 scrapes per day
- 9 non-vvm metrics
- 40 vvm metrics (38 non-partitioned and 2 partitioned)
- 1 node, 3 apps x 10 partitions:
  - values per day: (1 [node] * 9 + 3 [apps] * 38 + 2 * 10 [partitions]) * 5760 = 823680
  - partition size: ?
- 2 worker + 3 dbs, 5 apps x 10 partitions
  - values per day: (5 [nodes] * 9 + 5 [apps] * (38 + 2 * 10 [partitions])) * 5760 = 1929600
  - partition size: ?
- 50 worker + 3 dbs, 5 apps x 10 partitions, 5 apps x 100 partitions
  - values per day: (50 [nodes] * 9 + 5 [apps] * (38 + 2 * 10 [partitions]) + 5
  [apps] * (38 + 2 * 100 [partitions])) * 5760 = 11116800
  - partition size: ?

https://cql-calculator.herokuapp.com/
```
CREATE TABLE metrics (app text, day_in_month int, metric_name text, timestamp bigint, node text, partition int, value double, PRIMARY KEY ((app, day_in_month), metric_name, timestamp, node))
```

# See Also
- [core-imetrics](https://github.com/heeus/core-imetrics/)
- [A&D CE Monitoring Requirements](https://dev.heeus.io/launchpad/#!19448)
- [Full list of node_exporter metrics](https://github.com/prometheus/node_exporter/blob/master/collector/fixtures/e2e-output.txt)