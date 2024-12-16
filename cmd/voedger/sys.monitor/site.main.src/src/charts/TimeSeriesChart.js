/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { Box } from '@mui/material';
import * as React from 'react';
import { useState, useEffect } from 'react';
import { LineChart, Line, CartesianGrid, XAxis, YAxis, ResponsiveContainer, Tooltip } from 'recharts';
import { Error, useDataProvider, useTranslate } from 'react-admin';
import MonCard from '../elements/MonCard';
import Switch from '@mui/material/Switch';
import Typography from '@mui/material/Typography';
import { useSelector, useDispatch } from 'react-redux'
import { toggleItem, ALL } from '../features/filters/filtersSlice'
import { FormatValue } from '../utils/Units';
import { ResMetrics } from '../data/Resources';
import { Bars } from 'svg-loaders-react'

// https://www.heavy.ai/blog/12-color-palettes-for-telling-better-stories-with-your-data
export const PaletteSpringPastels = ["#fd7f6f", "#7eb0d5", "#b2e061", "#bd7ebe", "#ffb55a", "#ffee65", "#beb9db", "#fdcce5", "#8bd3c7"]
export const PaletteDutchField = ["#e60049", "#0bb4ff", "#50e991", "#e6d800", "#9b19f5", "#ffa300", "#dc0ab4", "#b3d4ff", "#00bfa0"]
export const PaletteStatusCodes = ["#7eb0d5", "#ffb55a", "#fd7f6f", "#e60049"]
export const LayoutLegendWidth = "300px"


/*
props:
    - caption
    - showAll
    - data: [
        {
            name: 'Serie 1'
            data: [
                {x: '10:11:12', value: 34},
                ...
            ]
        }, ...
    ]
    - aggs: ['avg']
    - path (to serialize filters)
    - units: 
        - percent
        - bps (bytes per second)

*/

function calcAgg(agg, entry, data, units, loading) {
    var v = 0
    if (loading) {
        return 0
    }
    if (agg === "avg") {
        data.forEach(e => {
            v += e[entry.id]
        })
        v = v/data.length
    }
    if (agg === "sum") {
        data.forEach(e => {
            v += e[entry.id]     
        })
    }

    return FormatValue(units, v)
}

const TimeSeriesChart = (props) => {

    const meta = props.meta
    const dataProvider = useDataProvider()


    const filter = useSelector((state) => state.filters.items[props.path] || [ALL])
    const appInterval = useSelector((state) => state.app.interval)

    const [loading, setLoading] = useState(true);
    const [error, setError] = useState();
    const [metrics, setMetrics] = useState(); 
    const [interval, setInterval] = useState(appInterval);

    const reload = (intv) => {
        const qmeta = meta.query
        qmeta.interval = intv
        dataProvider.getMany(ResMetrics, {meta: qmeta})
            .then(({ data }) => {
                setMetrics(data);
                setLoading(false);
            })
            .catch(error => {
                setError(error);
                setLoading(false);
            })
        }

    useEffect(() => {
        reload(interval)
    }, []);
    
    if (appInterval != interval) { // Interval has changed
        setInterval(appInterval)
        setLoading(true)
        setError(false)
        reload(appInterval);
    }

    const dispatch = useDispatch()
    const translate = useTranslate();
    
    if (error) {
        return (
            <MonCard caption={props.caption}>
                <Error />
            </MonCard>
        )
    }

    const data = loading?null:meta.transform(metrics)
    const colors = props.palette || PaletteSpringPastels
    
    const hdrStyle =  (props.aggs && props.aggs.length>0) ? {
        borderRight: '1px solid #ccc',
        paddingRight: '.7em',
    }:{}
    const cellStyle =  (props.aggs && props.aggs.length>0) ? {
        paddingRight: '.7em',
    }:{}
    const hdrStyle2 =  (props.aggs && props.aggs.length>0) ? {
        paddingLeft: '.5em',
    }:{}

    const filters = meta.dataKeys.flatMap(v => v.id)
    var moment = require('moment');

    return (
    <MonCard caption={props.caption} noframe={props.noframe}>
        <Box display="flex">
            {!props.nolegend?(
                <Box sx={{flexBasis: LayoutLegendWidth, flexShrink: 0, flexGrow: 0}}>
                <table cellPadding={1}><tbody>
                    {props.showAll?(
                    <tr key={"tr0"}>
                        <td><Switch size="small" disabled={loading} checked={filter.includes(ALL)} onChange={() => {dispatch(toggleItem({target: props.path, item: ALL, values: filters}))}} /></td>
                        <td style={hdrStyle}><Typography color="info" whiteSpace={'nowrap'}>{translate('common.showAll')}</Typography></td>
                        {props.aggs.map((agg, index) => (
                            <td key={`tr0td${index}`} style={hdrStyle2}><Typography whiteSpace={'nowrap'} color="info">{agg}</Typography></td>
                        ))}
                    </tr>):""}
                    {meta.dataKeys.map((entry, index) => {
                        const serieId = entry.id
                        return (
                            <tr key={`tr${index}`}>
                                <td><Switch size="small" checked={filter.includes(ALL) || filter.includes(serieId)} onChange={() => {
                                    dispatch(toggleItem({target: props.path, item: serieId, values: filters}))
                                }}/></td>
                                <td style={cellStyle}><Typography whiteSpace={'nowrap'} color={colors[index]}>{entry.name}</Typography></td>
                                {
                                    props.aggs.map((agg, i2) => (
                                        <td key={`tr${index}td${i2}`} style={hdrStyle2}><Typography whiteSpace={'nowrap'} color={colors[index]}>{calcAgg(agg, entry, data, props.units, loading)}</Typography></td>
                                    ))
                                }
                            </tr>
                        )
                    })}
                    </tbody></table>
                </Box>):""}
            {!props.nochart?(
                <Box sx={{flex: 1}}>

                    {loading?(
                        <Box width="100%" height={props.height} sx={{ border: '1px solid #ccc' }} display='flex' alignItems={'center'} justifyContent={'center'}>
                            <Bars stroke={PaletteSpringPastels[1]} fill={PaletteSpringPastels[1]} width="60"/>
                        </Box>
                    ):(
                        <ResponsiveContainer width="95%" aspect={props.aspect} height={props.height}>
                            <LineChart data={data} style={{alignSelf: "center"}} width={500} height={200}  margin={{ top: 5, right: 20, bottom: 5, left: 10 }}>
                                {meta.dataKeys.map((entry, index) => {
                                    if (filter.includes(entry.id) || filter.includes(ALL)) {
                                        return(
                                            <Line key={`line${index}`} dot={false} isAnimationActive={false} name={entry.name} type="monotone" dataKey={entry.id} stroke={colors[index]} activeDot={{ r: 8 }} />
                                        )
                                    }
                                    return ""
                                })}
                                <CartesianGrid stroke="#ccc" strokeDasharray="3 3"/>
                                <Tooltip 
                                    labelFormatter={(lbl) => {return moment(lbl).format('HH:mm:ss')}} 
                                    formatter={(v) => {return FormatValue(props.units, v)}} />
                                <XAxis dataKey="x" type='number' scale='time' domain = {['dataMin', 'dataMax']} tickFormatter = {(unixTime) => moment(unixTime).format('HH:mm:ss')} />
                                <YAxis tickFormatter={(v) => {return FormatValue(props.units, v)}} />
                            </LineChart>
                        </ResponsiveContainer>
                    )}
                </Box>
            ):""}
        </Box>
        {props.footer}
    </MonCard>
    )
};

export default TimeSeriesChart
// <XAxis dataKey="x" tickFormatter={timeStr => moment(timeStr).format('HH:mm')} allowDuplicatedCategory={false} interval={20} />
 