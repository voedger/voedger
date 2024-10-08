/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { Title } from 'react-admin';
import { PaletteStatusCodes } from '../layout/Palette';
import { Box } from '@mui/material';
import { useTranslate } from 'react-admin';
import TimeSeriesChart from '../charts/TimeSeriesChart';
import { HttpStatusCodesMeta, ProcessorsPerformanceRpsMeta, StorageIopsCacheHitsMeta, StorageIopsMeta, SysApp } from '../data/Resources';
import SysPerformanceWorstApps from './SysPerformanceWorstApps';
import MonCard from './MonCard';

const SysPerformance = (props) => {

    const translate = useTranslate();

    return (
        <Box>
            <Title title={props.title} />
            <TimeSeriesChart 
                path={props.path+":rps"} 
                caption={translate('charts.rps')} 
                meta={ProcessorsPerformanceRpsMeta(SysApp)}
                aggs={['avg']} 
                units={'count'}
                height={200}
                showAll />
            <TimeSeriesChart 
                path={props.path+":statusCodes"} 
                caption={translate('charts.statusCodes')} 
                meta={HttpStatusCodesMeta(SysApp)}
                aggs={['sum']} 
                units={'count'}
                palette={PaletteStatusCodes}
                height={200}
                showAll
                summaryChart />
            <MonCard caption={translate('charts.iops')}>
                <TimeSeriesChart 
                    noframe
                    caption={translate('sysPerformance.executionTime')} 
                    path={props.path+":iops"}                 
                    meta={StorageIopsMeta(SysApp)}
                    aggs={['avg']} 
                    units={'count'}
                    height={200}
                    showAll />
                <TimeSeriesChart 
                    noframe
                    caption={translate('sysPerformance.cacheHits')} 
                    path={props.path+":cacheHits"}                 
                    meta={StorageIopsCacheHitsMeta(SysApp)}
                    aggs={['avg']} 
                    units={'count'}
                    height={200}
                    showAll />
                <SysPerformanceWorstApps />
            </MonCard>
        </Box>
    )
};

export default SysPerformance