/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { Title } from 'react-admin';
import { useTranslate } from 'react-admin';
import { BPS, COUNT, PERCENT } from '../utils/Units';
import { HvmsCpuUsageMeta, HvmsDiskIOMeta, HvmsDiskUsageMeta, HvmsIOPSMeta, HvmsMemoryUsageMeta, SysApp } from '../data/Resources';
import TimeSeriesChart from '../charts/TimeSeriesChart';

const SysResources = (props) => {

    const translate = useTranslate();

    const hvms = ['worker', 'db1', 'db2', 'db3']

    return (
        <div>
            <Title title={props.title} />
            <TimeSeriesChart 
                path={props.path+":cpu"} 
                caption={translate('charts.cpuUsage')} 
                meta={HvmsCpuUsageMeta(SysApp, hvms)}
                aggs={['avg']} 
                units={PERCENT}
                height={200}
                showAll />

            <TimeSeriesChart 
                path={props.path+":memory"} 
                caption={translate('charts.memUsage')} 
                meta={HvmsMemoryUsageMeta(SysApp, hvms)}
                aggs={['avg']} 
                units={PERCENT}
                height={200}
                showAll />

            <TimeSeriesChart 
                path={props.path+":disk"} 
                caption={translate('charts.diskUsage')} 
                meta={HvmsDiskUsageMeta(SysApp, hvms)}
                aggs={['avg']} 
                units={PERCENT}
                height={200}
                showAll />

            <TimeSeriesChart 
                path={props.path+":diskIo"} 
                caption={translate('charts.diskIO')} 
                meta={HvmsDiskIOMeta(SysApp, hvms)}
                aggs={['avg']} 
                units={BPS}
                height={200}
                showAll />

            <TimeSeriesChart 
                path={props.path+":iops"} 
                caption={translate('charts.iops')} 
                meta={HvmsIOPSMeta(SysApp, hvms)}
                aggs={['avg']} 
                units={COUNT}
                height={200}
                showAll />
        </div>
    )
};

export default SysResources