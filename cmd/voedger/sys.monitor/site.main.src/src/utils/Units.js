/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */
import {filesize} from "filesize";
import { AUTH_CHECK } from "react-admin";

export const PERCENT = "percent"
export const BPS = "bps"
export const DATASIZE = "datasize"
export const COUNT = "count"
export const DURATION = "duration" // from Nanoseconds

export function Rnd(min, max) {
    return min + Math.floor(Math.random() * (max - min));
}

const EMULATE_SECONDS = 3600
const EMULATE_STEP_SEC = 15
export const NANOS_IN_SECOND = 1000000000

/*
    in : [
        {
            key: string
            start: int
            offs: 10
        }, 
        ...
    ],

    out: [
        {
            x: Date,
            key1: value1_1,
            key2: value1_2,
            ...
        },
        ...
    ]

*/

export function EmuData(meta) {
    const now = new Date()
    const before = new Date(now.getTime() - EMULATE_SECONDS * 1000)
    var t = new Date(before.getTime())
    var result = []
    var last = null
    while (t < now) {
        var cur = {}
        meta.map((e) => {
            cur.x = new Date(t.getTime())
            if (e.calc) {
                cur[e.key] = e.calc(cur)    
            } else {
                cur[e.key] = last ? Math.max(0, last[e.key] + Rnd(-e.offs, e.offs)) : e.start
            }
        })
        result.push(cur)
        last = cur        
        t.setSeconds(t.getSeconds() + EMULATE_STEP_SEC)
    }
    return result
}

export function EmuSerie(id, name, start, offs) {
    const now = new Date()
    const before = new Date(now.getTime() - EMULATE_SECONDS * 1000)
    var t = new Date(before.getTime())
    var serie = {
        id: id, 
        name: name,
        data: []
    }
    var last = null
    while (t < now) {
        var cur = {
            x: t.getTime(),
            value: last ? Math.max(0, last.value + Rnd(-offs, offs)) : start
        }
        serie.data.push(cur)
        last = cur        
        t.setSeconds(t.getSeconds() + EMULATE_STEP_SEC)
    }
    return serie
}

export function EmulateData(callback) {
    const now = new Date()
    const before = new Date(now.getTime() - EMULATE_SECONDS * 1000)
    var t = new Date(before.getTime())
    var result = []
    var last = null
    while (t < now) {
        var cur = callback(new Date(t.getTime()), last)
        result.push(cur)
        last = cur        
        t.setSeconds(t.getSeconds() + EMULATE_STEP_SEC)
    }
    return result
}

export function FormatValue(units, value) {
    if (units === PERCENT) {
        return Math.floor(value) + "%"
    }
    if (units === BPS) {
        return filesize(value) + "s"
    }
    if (units === DATASIZE) {
        return filesize(value)
    }
    if (units === COUNT) {
        var hrnumbers = require('human-readable-numbers');
        return hrnumbers.toHumanString(value)
    }
    if (units === DURATION) {
        return nanosecondsToStr(value)
    }
    return value
}

function nanosecondsToStr(nanos) {
    if (nanos >= 1000) {
        return microsecondsToStr(nanos / 1000)
    } else {
        return Math.round(nanos*10)/10 + ' ns'
    }
}

function microsecondsToStr(micros) {
    if (micros >= 1000) {
        return millisecondsToStr(micros / 1000)
    } else {
        return Math.round(micros*10)/10 + ' us'
    }
}

function millisecondsToStr(millis) {
    if (millis >= 1000) {
        return secondsToStr(millis / 1000)
    } else {
        return Math.round(millis*10)/10 + ' ms'
    }
}

function secondsToStr(seconds) {

    var years = Math.floor(seconds / 31536000);
    if (years) {
        return years + ' yr'
    }
    var days = Math.floor((seconds %= 31536000) / 86400);
    if (days) {
        return days + ' d';
    }
    var hours = Math.floor((seconds %= 86400) / 3600);
    if (hours) {
        return hours + ' h';
    }
    var minutes = Math.floor((seconds %= 3600) / 60);
    if (minutes) {
        return minutes + ' m';
    }

    return Math.round(seconds*10)/10 + ' s';
}
