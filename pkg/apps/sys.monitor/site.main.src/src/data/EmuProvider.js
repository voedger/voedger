/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { ResMetrics, ResWorstApps } from "./Resources";

const SCRAPE_INTVL = 15

const emuValues = {
  ///////////////////////////////////
  //  HTTP
  heeus_http_status_2xx_total: 200 * SCRAPE_INTVL,
  heeus_http_status_4xx_total: 50 * SCRAPE_INTVL,
  heeus_http_status_5xx_total: 50 * SCRAPE_INTVL,
  heeus_http_status_503_total: 20 * SCRAPE_INTVL,
  ///////////////////////////////////
  //  IStorageCache
  heeus_istoragecache_get_total: 100 * SCRAPE_INTVL,
  heeus_istoragecache_get_cached_total: {
    value: 35 * SCRAPE_INTVL,
    max: 'heeus_istoragecache_get_total',
  },
  heeus_istoragecache_getbatch_total: 120 * SCRAPE_INTVL,
  heeus_istoragecache_getbatch_cached_total: {
    value: 60 * SCRAPE_INTVL,
    max: 'heeus_istoragecache_getbatch_total',
  },
  heeus_istoragecache_put_total: 50 * SCRAPE_INTVL,
  heeus_istoragecache_putbatch_total: 60 * SCRAPE_INTVL,
  heeus_istoragecache_read_total: 80 * SCRAPE_INTVL,
  heeus_istoragecache_get_seconds: 2 * SCRAPE_INTVL,
  heeus_istoragecache_getbatch_seconds: 7 * SCRAPE_INTVL,
  heeus_istoragecache_put_seconds: 3 * SCRAPE_INTVL,
  heeus_istoragecache_putbatch_seconds: 12 * SCRAPE_INTVL,
  heeus_istoragecache_read_seconds: 6 * SCRAPE_INTVL,
  
  ///////////////////////////////////
  //  Command Processor
  heeus_cp_commands_total: 100 * SCRAPE_INTVL,
  heeus_cp_commands_seconds: 2 * SCRAPE_INTVL,
  heeus_cp_exec_seconds: 1 * SCRAPE_INTVL,
  heeus_cp_validate_seconds: 0.5 * SCRAPE_INTVL,
  heeus_cp_putplog_seconds: 0.5 * SCRAPE_INTVL,
  
  ///////////////////////////////////
  //  Query Processor
  heeus_qp_queries_total: 200 * SCRAPE_INTVL,
  heeus_qp_queries_seconds: 4  * SCRAPE_INTVL,
  heeus_qp_build_seconds: 1  * SCRAPE_INTVL,
  heeus_qp_exec_seconds: 3 * SCRAPE_INTVL,
  heeus_qp_exec_fields_seconds: 0.5 * SCRAPE_INTVL,
  heeus_qp_exec_enrich_seconds: 0.5 * SCRAPE_INTVL,
  heeus_qp_exec_filter_seconds: 0.5 * SCRAPE_INTVL,
  heeus_qp_exec_order_seconds: 0.5 * SCRAPE_INTVL,
  heeus_qp_exec_count_seconds: 0.5 * SCRAPE_INTVL,
  heeus_qp_exec_send_seconds: 0.5 * SCRAPE_INTVL,

  ///////////////////////////////////
  //  Node Metrics
  node_cpu_idle_seconds_total: 0.2 * SCRAPE_INTVL,
  node_memory_memavailable_bytes: {
    value: 32000000000,
    gauge: true,
    max: 64000000000,
    min: 3000000000,  
    offs: 100000000,
  },
  node_memory_memtotal_bytes: {
    value: 64000000000,
    fixed: true,
  },
  node_filesystem_free_bytes: {
    value: 2000000000000,
    gauge: true,
    max: 4000000000000,
    min: 10000000000,  
    offs: 800000000,
  },
  node_filesystem_size_bytes: {
    value: 4000000000000,
    fixed: true,
  },
  node_disk_read_bytes_total: 100000,
  node_disk_write_bytes_total: 50000,
  node_disk_reads_completed_total: 20000,
  node_disk_writes_completed_total: 10000,
}

const EMULATE_LOADING_SEC = 1

export function rnd(min, max) {
  return min + Math.random() * (max - min);
}

export function EmuMetrics(app, metrics, interval, hvms) {
  var result = []
  const now = new Date()
  const before = new Date(now.getTime() - interval * 1000)
  var t = new Date(before.getTime())
  var values = {}
  var offsets = {}
  if (!hvms) {
    hvms = [""]
  }
  while (t < now) {
      metrics.map((metric) => {
        const emuVal = emuValues[metric] 
        var init = emuVal
        var min = null
        var max = null
        var offs = null
        var fixed = false
        var gauge = false
        if (typeof emuVal !== 'number') {
          init = emuVal.value
          min = emuVal.min
          max = emuVal.max
          if (typeof emuVal.offs === 'number') {
            offs = emuVal.offs
          } 
          fixed = emuVal.fixed
          gauge = emuVal.gauge
        }
        if (!offs) {
          offs = init / 5
        }


        hvms.map((hvm) => {
          const keyprefix = hvm + "__"
          const key = keyprefix + metric

          if (values[key]===undefined || fixed) {
              values[key] = init
          } else {
              if (offsets[key]===undefined) {
                  offsets[key] = gauge ? 1 : offs
              } else {
                  if (gauge) {
                    offsets[key] = offsets[key]+rnd(-offs, offs)
                  } else {
                    offsets[key] = Math.max(0, offsets[key]+rnd(-offs, offs))
                  }
              }

              if (gauge) {
                values[key] = values[key] + offsets[key]
                if (typeof max === 'number' && values[key] > max) {
                  values[key] = max
                  offsets[key] = 0
                }  
                if (typeof min === 'number' && values[key] < min) {
                  values[key] = min
                  offsets[key] = 0
                }  
              } else  { // counter
                  if (typeof max === 'string' && offsets[key] > offsets[keyprefix+max]) {
                    offsets[key] = offsets[keyprefix+max]
                  } else if (typeof max === 'number' && offsets[key] > max) {
                    offsets[key] = max
                  }  
                  values[key] = values[key] + offsets[key]
              }
            }

          

          var mv = {
              id: 0,
              time: t.getTime(),
              metric: metric,
              hvm: hvm,
              app: app,
              value: values[key],
          }
          result.push(mv)
          return true
        })
      })
      t.setSeconds(t.getSeconds() + SCRAPE_INTVL)
  }
  return result
}

function EmuWorstApps() {
  return {
    moreGetTop5AppsByRT : [
      {app: 'untill/air', value: 940},
      {app: 'sys/monitor', value: 20430},
      {app: 'sys/registry', value: 2114640},
    ],
    moreGetTop5AppsByRPS: [
      {app: 'untill/air', value: 12034},
      {app: 'sys/registry', value: 456},
      {app: 'sys/monitor', value: 67},
    ],
    moreGetBottom5AppsByCacheHits: [
      {app: 'untill/air', value: 45},
      {app: 'sys/registry', value: 32},
      {app: 'sys/monitor', value: 8},
    ],
    moreGetBatchTop5AppsByRT : [
      {app: 'untill/air', value: 30},
      {app: 'sys/monitor', value: 41},
      {app: 'sys/registry', value: 62},
    ],
    moreGetBatchTop5AppsByRPS : [
      {app: 'untill/air', value: 12034},
      {app: 'sys/registry', value: 456},
      {app: 'sys/monitor', value: 67},
    ],
    moreGetBatchBottom5AppsByCacheHits : [
      {app: 'untill/air', value: 45},
      {app: 'sys/registry', value: 32},
      {app: 'sys/monitor', value: 8},
    ],
    morePutTop5AppsByRT : [
      {app: 'untill/air', value: 940},
      {app: 'sys/monitor', value: 20430},
      {app: 'sys/registry', value: 2114640},
    ],
    morePutTop5AppsByRPS: [
      {app: 'untill/air', value: 12034},
      {app: 'sys/registry', value: 456},
      {app: 'sys/monitor', value: 67},
    ],
    moreReadTop5AppsByRT : [
      {app: 'untill/air', value: 940},
      {app: 'sys/monitor', value: 20430},
      {app: 'sys/registry', value: 2114640},
    ],
    moreReadTop5AppsByRPS: [
      {app: 'untill/air', value: 12034},
      {app: 'sys/registry', value: 456},
      {app: 'sys/monitor', value: 67},
    ],
    morePutBatchTop5AppsByRT : [
      {app: 'untill/air', value: 940},
      {app: 'sys/monitor', value: 20430},
      {app: 'sys/registry', value: 2114640},
    ],
    morePutBatchTop5AppsByRPS: [
      {app: 'untill/air', value: 12034},
      {app: 'sys/registry', value: 456},
      {app: 'sys/monitor', value: 67},
    ],
    morePutBatchTop5AppsByBatchSize: [
      {app: 'untill/air', value: 60},
      {app: 'sys/registry', value: 12},
      {app: 'sys/monitor', value: 6},
    ]
  }
}

export default {
    getList: (resource, params) => {
    },

    getOne: (resource, params) => {
      return new Promise((resolve, reject) => {
        setTimeout(() => {
            var emuData
            switch (resource) {
              case ResWorstApps: 
                emuData = EmuWorstApps();
                break;              
              default:
                emuData = null;
            }
            emuData.id = 0
            resolve({
              data: emuData
            })
          }, rnd(EMULATE_LOADING_SEC - EMULATE_LOADING_SEC/4, EMULATE_LOADING_SEC + EMULATE_LOADING_SEC/4)*1000);
        });
    },

    getMany: (resource, params) => {
      return new Promise((resolve, reject) => {
        const { app, metrics, interval, hvms } = params.meta;
        setTimeout(() => {
            var emuData
            switch (resource) {
              case ResMetrics: 
                emuData = EmuMetrics(app, metrics, interval, hvms);
                break;              
              default:
                emuData = null;
            }
            resolve({
              data: emuData
            })
          }, rnd(EMULATE_LOADING_SEC - EMULATE_LOADING_SEC/4, EMULATE_LOADING_SEC + EMULATE_LOADING_SEC/4)*1000);
        });
  },

    getManyReference: () => {},

    create: () => {},

    update: () => {},

    updateMany: () => {},

    delete: () => {},

    deleteMany: () => {},
};
