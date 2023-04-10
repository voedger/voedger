/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { NANOS_IN_SECOND } from "../utils/Units"

/*
out:
    [
        {
            x: time1,
            q: uint64,
            c: uint64,
            tot: uint64,
        },
        ...
    ]

*/
export const ResMetrics = "metrics"
export const ResWorstApps = "worstapps"
export const SysApp = "sys"
const MILLIS_IN_SECOND = 1000

function rate(e0, e1, key) {
    if (!e0) {
        return 0
    }
    const seconds = (e1.time - e0.time) / MILLIS_IN_SECOND    
    if (seconds == 0) {
        return 0
    }
    const rate = (e1[key] - e0[key]) / seconds
    return rate
}

function diff(e0, e1, key) {
    if (!e0) {
        return 0
    }
    return (e1[key] - e0[key])
}

function exectime(prev, cur, keySeconds, keyTotal) {
    if (prev == 0) {
        return 0
    }
    const total = diff(prev, cur, keyTotal)
    if (total == 0) {
        return 0
    }
    const seconds = diff(prev, cur, keySeconds)/total
    return seconds * NANOS_IN_SECOND

}



/*
    in:
        - metrics: [
            {time: uint64, metric: string, hvm: string, value: uint64},
            ...
        ]
        - callback: func()

    out: [
        {
            time: uint64,
            metric1: uint64, // or hvm1_metric1: uint64
            metric2: uint64, // or hvm1_metric2: uint64
        }
    ]

*/
function transformTimeSeries(metrics, callback) {
    var result = []
    var timeUnit = {};
    var prevTimeUnit = null

    const flush = function() {
        var out = callback(timeUnit, prevTimeUnit)
        out.x = timeUnit.time
        result.push(out)
        prevTimeUnit = timeUnit
        timeUnit = {}
    }

    metrics.map((e) => {
        if (timeUnit.time !== e.time) {
            if (timeUnit.time) {
                flush()
            }
            timeUnit.time = e.time
        }
        const key = (e.hvm == "")?e.metric:`${e.hvm}__${e.metric}`
        timeUnit[key] = e.value
    })
    if (timeUnit.time) {
        flush()
    }

    return result
}

export function HvmsCpuUsageMeta(appName, hvms) {
    var dataKeys = []
    hvms.map((hvm) => {
        dataKeys.push({
            id: hvm,
            name: hvm,
        })
        return true
    })
    return  {
        query: {
            metrics: ['node_cpu_idle_seconds_total'], 
            app: appName,
            hvms: hvms,
        },
        dataKeys: dataKeys,
        transform: (metrics) => {
            return transformTimeSeries(metrics, (src, srcPrev) => {
                var obj = {}
                hvms.map((hvm) => {
                    const value = Math.floor(rate(srcPrev, src, `${hvm}__node_cpu_idle_seconds_total`)*100)
                    obj[hvm] = value
                })
                return obj
            })
        }
    }
}

export function HvmsMemoryUsageMeta(appName, hvms) {
    var dataKeys = []
    hvms.map((hvm) => {
        dataKeys.push({
            id: hvm,
            name: hvm,
        })
        return true
    })
    return  {
        query: {
            metrics: ['node_memory_memavailable_bytes', 'node_memory_memtotal_bytes'], 
            app: appName,
            hvms: hvms,
        },
        dataKeys: dataKeys,
        transform: (metrics) => {
            return transformTimeSeries(metrics, (src, srcPrev) => {
                var obj = {}
                hvms.map((hvm) => {
                    obj[hvm] = src[`${hvm}__node_memory_memavailable_bytes`] / src[`${hvm}__node_memory_memtotal_bytes`] * 100
                })
                return obj
            })
        }
    }
}

export function HvmsDiskUsageMeta(appName, hvms) {
    var dataKeys = []
    hvms.map((hvm) => {
        dataKeys.push({
            id: hvm,
            name: hvm,
        })
        return true
    })
    return  {
        query: {
            metrics: ['node_filesystem_free_bytes', 'node_filesystem_size_bytes'], 
            app: appName,
            hvms: hvms,
        },
        dataKeys: dataKeys,
        transform: (metrics) => {
            return transformTimeSeries(metrics, (src, srcPrev) => {
                var obj = {}
                hvms.map((hvm) => {
                    obj[hvm] = src[`${hvm}__node_filesystem_free_bytes`] / src[`${hvm}__node_filesystem_size_bytes`] * 100
                })
                return obj
            })
        }
    }
}

export function HvmsDiskIOMeta(appName, hvms) {
    var dataKeys = []
    hvms.map((hvm) => {
        dataKeys.push({
            id: hvm,
            name: hvm,
        })
        return true
    })
    return  {
        query: {
            metrics: ['node_disk_read_bytes_total', 'node_disk_write_bytes_total'], 
            app: appName,
            hvms: hvms,
        },
        dataKeys: dataKeys,
        transform: (metrics) => {
            return transformTimeSeries(metrics, (src, srcPrev) => {
                var obj = {}
                hvms.map((hvm) => {
                    obj[hvm] = rate(srcPrev, src, `${hvm}__node_disk_read_bytes_total`) + rate(srcPrev, src, `${hvm}__node_disk_write_bytes_total`)
                })
                return obj
            })
        }
    }
}

export function HvmsIOPSMeta(appName, hvms) {
    var dataKeys = []
    hvms.map((hvm) => {
        dataKeys.push({
            id: hvm,
            name: hvm,
        })
        return true
    })
    return  {
        query: {
            metrics: ['node_disk_reads_completed_total', 'node_disk_writes_completed_total'], 
            app: appName,
            hvms: hvms,
        },
        dataKeys: dataKeys,
        transform: (metrics) => {
            return transformTimeSeries(metrics, (src, srcPrev) => {
                var obj = {}
                hvms.map((hvm) => {
                    obj[hvm] = rate(srcPrev, src, `${hvm}__node_disk_writes_completed_total`) + rate(srcPrev, src, `${hvm}__node_disk_reads_completed_total`)
                })
                return obj
            })
        }
    }
}

export function ProcessorsPerformanceRpsMeta(appName) {
    return  {
        query: {
            metrics: ['heeus_cp_commands_total', 'heeus_qp_queries_total'], 
            app: appName,
        },
        dataKeys: [
            {id: "q", name: "Queries"},
            {id: "c", name: "Commands"},
            {id: "tot", name: "Total"},
        ],
        transform: (metrics) => {
            return transformTimeSeries(metrics, (src, srcPrev) => {
                const qq = Math.floor(rate(srcPrev, src, "heeus_qp_queries_total"))
                const cc = Math.floor(rate(srcPrev, src, "heeus_cp_commands_total"))
                return {
                    q: qq,
                    c: cc,
                    tot: cc+qq,
                }
            })
        }
    }
}

export function HttpStatusCodesMeta(appName) {
    return  {
        query: {
            metrics: ['heeus_http_status_2xx_total', 'heeus_http_status_4xx_total', 'heeus_http_status_5xx_total', 'heeus_http_status_503_total'], 
            app: appName,
        },
        dataKeys: [
            {id: "c2xx", name: "2xx"},
            {id: "c4xx", name: "4xx"},
            {id: "c5xx", name: "5xx"},
            {id: "c503", name: "503"},
        ],
        transform: (metrics) => {
            return transformTimeSeries(metrics, (src, srcPrev) => {
                return {
                    c2xx: Math.floor(diff(srcPrev, src, "heeus_http_status_2xx_total")),
                    c4xx: Math.floor(diff(srcPrev, src, "heeus_http_status_4xx_total")),
                    c5xx: Math.floor(diff(srcPrev, src, "heeus_http_status_5xx_total")),
                    c503: Math.floor(diff(srcPrev, src, "heeus_http_status_503_total")),
                }
            })
        }
    }
}

export function StorageIopsMeta(appName) {
    return  {
        query: {
            metrics: ['heeus_istoragecache_get_total', 
                'heeus_istoragecache_getbatch_total', 'heeus_istoragecache_put_total', 
                'heeus_istoragecache_putbatch_total', 'heeus_istoragecache_read_total'], 
            app: appName,
        },
        dataKeys: [
            {id: "get", name: "Get"},
            {id: "getbatch", name: "GetBatch"},
            {id: "put", name: "Put"},
            {id: "putbatch", name: "PutBatch"},
            {id: "read", name: "Read"},
        ],
        transform: (metrics) => {
            return transformTimeSeries(metrics, (src, srcPrev) => {
                return {
                    get: Math.floor(rate(srcPrev, src, "heeus_istoragecache_get_total")),
                    getbatch: Math.floor(rate(srcPrev, src, "heeus_istoragecache_getbatch_total")),
                    put: Math.floor(rate(srcPrev, src, "heeus_istoragecache_put_total")),
                    putbatch: Math.floor(rate(srcPrev, src, "heeus_istoragecache_putbatch_total")),
                    read: Math.floor(rate(srcPrev, src, "heeus_istoragecache_read_total")),
                }
            })
        }
    }
}

export function StorageIopsCacheHitsMeta(appName) {
    return  {
        query: {
            metrics: ['heeus_istoragecache_get_total', 'heeus_istoragecache_get_cached_total',
                'heeus_istoragecache_getbatch_total', 'heeus_istoragecache_getbatch_cached_total'],
            app: appName,
        },
        dataKeys: [
            {id: "get", name: "Get"},
            {id: "getBatch", name: "GetBatch"},
        ],
        transform: (metrics) => {
            return transformTimeSeries(metrics, (cur, prev) => {
                if (!prev) {
                    return {
                        get: 0,
                        getBatch: 0,
                    }
                }
                const dg = diff(prev, cur, "heeus_istoragecache_get_total")
                const dgc = diff(prev, cur, "heeus_istoragecache_get_cached_total")
                
                const dgb = diff(prev, cur, "heeus_istoragecache_getbatch_total")
                const dgbc = diff(prev, cur, "heeus_istoragecache_getbatch_cached_total")

                const res = {
                    get: dg!=0 ? dgc/dg * 100 : 0,
                    getBatch: dgb!=0 ? dgbc/dgb * 100 : 0
                }

                return res
            })
        }
    }
}

export function StorageIopsExecutionTimeMeta(appName) {
    return  {
        query: {
            metrics: ['heeus_istoragecache_get_seconds', 'heeus_istoragecache_get_total',
                        'heeus_istoragecache_getbatch_seconds', 'heeus_istoragecache_getbatch_total', 
                        'heeus_istoragecache_put_seconds', 'heeus_istoragecache_put_total', 
                        'heeus_istoragecache_putbatch_seconds', 'heeus_istoragecache_putbatch_total', 
                        'heeus_istoragecache_read_seconds', 'heeus_istoragecache_read_total', 
                    ], 
            app: appName,
        },
        dataKeys: [
            {id: "get", name: "Get"},
            {id: "getbatch", name: "GetBatch"},
            {id: "put", name: "Put"},
            {id: "putbatch", name: "PutBatch"},
            {id: "read", name: "Read"},
        ],
        transform: (metrics) => {
            return transformTimeSeries(metrics, (cur, prev) => {
                return {
                    get: exectime(prev, cur, "heeus_istoragecache_get_seconds", "heeus_istoragecache_get_total"),
                    getbatch: exectime(prev, cur, "heeus_istoragecache_getbatch_seconds", "heeus_istoragecache_getbatch_total"),
                    put: exectime(prev, cur, "heeus_istoragecache_put_seconds", "heeus_istoragecache_put_total"),
                    putbatch: exectime(prev, cur, "heeus_istoragecache_putbatch_seconds", "heeus_istoragecache_putbatch_total"),
                    read: exectime(prev, cur, "heeus_istoragecache_read_seconds", "heeus_istoragecache_read_total"),
                }
            })
        }
    }
}

export function CommandProcessorMeta(appName) {
    return  {
        query: {
            metrics: ['heeus_cp_commands_total', 
                'heeus_cp_commands_seconds', 'heeus_cp_exec_seconds', 
                'heeus_cp_validate_seconds', 'heeus_cp_putplog_seconds'], 
            app: appName,
        },
        dataKeys: [
            {id: "tot", name: "Total Command"},
            {id: "validate", name: "Validate"},
            {id: "exec", name: "Exec"},
            {id: "plog", name: "PutPLog"},
        ],
        transform: (metrics) => {
            return transformTimeSeries(metrics, (cur, prev) => {
                return {
                    tot: exectime(prev, cur, "heeus_cp_commands_seconds", "heeus_cp_commands_total"),
                    validate: exectime(prev, cur, "heeus_cp_validate_seconds", "heeus_cp_commands_total"),
                    exec: exectime(prev, cur, "heeus_cp_exec_seconds", "heeus_cp_commands_total"),
                    plog: exectime(prev, cur, "heeus_cp_putplog_seconds", "heeus_cp_commands_total"),
                }
            })
        }
    }
}

export function QueryProcessorMeta(appName) {
    return  {
        query: {
            metrics: ['heeus_qp_queries_total', 
                'heeus_qp_queries_seconds', 'heeus_qp_build_seconds', 
                'heeus_qp_exec_seconds', 'heeus_qp_exec_fields_seconds',
                'heeus_qp_exec_enrich_seconds', 'heeus_qp_exec_filter_seconds',
                'heeus_qp_exec_order_seconds','heeus_qp_exec_count_seconds',
                'heeus_qp_exec_send_seconds'], 
            app: appName,
        },
        dataKeys: [
            {id: "tot", name: "Total Query"},
            {id: "build", name: "Build"},
            {id: "exec", name: "Exec"},
            {id: "execFields", name: "Exec/Fields"},
            {id: "execEnrich", name: "Exec/Enrich"},
            {id: "execFilter", name: "Exec/Filter"},
            {id: "execOrder", name: "Exec/Order"},
            {id: "execCount", name: "Exec/Count"},
            {id: "execSend", name: "Exec/Send"},
        ],
        transform: (metrics) => {
            return transformTimeSeries(metrics, (src, srcPrev) => {
                const queries = diff(srcPrev, src, "heeus_qp_queries_total")
                
                if (queries == 0) {
                    return {
                        tot: 0,
                        build: 0,
                        exec: 0,
                        execFields: 0,
                        execEnrich: 0,
                        execFilter: 0,
                        execOrder: 0,
                        execCount: 0,
                        execSend: 0,
                    }
                }

                const tq = diff(srcPrev, src, "heeus_qp_queries_seconds")
                const tb = diff(srcPrev, src, "heeus_qp_build_seconds")
                const te = diff(srcPrev, src, "heeus_qp_exec_seconds")
                const tefld = diff(srcPrev, src, "heeus_qp_exec_fields_seconds")
                const tee = diff(srcPrev, src, "heeus_qp_exec_enrich_seconds")
                const teflt = diff(srcPrev, src, "heeus_qp_exec_filter_seconds")
                const teo = diff(srcPrev, src, "heeus_qp_exec_order_seconds")
                const tec = diff(srcPrev, src, "heeus_qp_exec_count_seconds")
                const tes = diff(srcPrev, src, "heeus_qp_exec_send_seconds")

                return {
                    tot: srcPrev ? (tq/queries) * NANOS_IN_SECOND : 0,
                    build: srcPrev ? (tb/queries) * NANOS_IN_SECOND : 0,
                    exec: srcPrev ? (te/queries) * NANOS_IN_SECOND : 0,
                    execFields: srcPrev ? (tefld/queries) * NANOS_IN_SECOND : 0,
                    execEnrich: srcPrev ? (tee/queries) * NANOS_IN_SECOND : 0,
                    execFilter: srcPrev ? (teflt/queries) * NANOS_IN_SECOND : 0,
                    execOrder: srcPrev ? (teo/queries) * NANOS_IN_SECOND : 0,
                    execCount: srcPrev ? (tec/queries) * NANOS_IN_SECOND : 0,
                    execSend: srcPrev ? (tes/queries) * NANOS_IN_SECOND : 0,
                }
            })
        }
    }
}


