/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { Title } from 'react-admin';
import Typography from '@mui/material/Typography';
import { PaletteStatusCodes } from '../layout/Palette';
import { useTranslate } from 'react-admin';
import { useSearchParams } from "react-router-dom";
import { COUNT, DURATION, PERCENT } from '../utils/Units';
import { CommandProcessorMeta, HttpStatusCodesMeta, ProcessorsPerformanceRpsMeta, QueryProcessorMeta, StorageIopsCacheHitsMeta, StorageIopsExecutionTimeMeta, StorageIopsMeta } from '../data/Resources';
import TimeSeriesChart, { LayoutLegendWidth } from '../charts/TimeSeriesChart';
import MonCard from './MonCard';
import AppSlowProjectors from './AppSlowProjectors';
import AppPartitionsBalance from './AppPartitionsBalance';
import { Box } from '@mui/material';


const AppPerformance = (props) => {

    let [searchParams] = useSearchParams();    
    const app = searchParams.get("app")
    const translate = useTranslate();

    return (
        <div>
            <Title title={translate('menu.appPerformance')+': '+app} />
            <TimeSeriesChart 
                path={props.path+":rps"} 
                caption={translate('charts.rps')} 
                meta={ProcessorsPerformanceRpsMeta(app)}
                aggs={['avg']} 
                units={COUNT}
                height={200}
                showAll />
            <MonCard caption={translate('charts.statusCodes')} >
                <TimeSeriesChart 
                    noframe
                    path={props.path+":statusCodes"} 
                    caption={translate('appPerformance.overall')} 
                    meta={HttpStatusCodesMeta(app)}
                    aggs={['sum']} 
                    units={COUNT}
                    palette={PaletteStatusCodes}
                    height={200}
                    showAll />
                <Box display='flex'>
                    <Box sx={{flexBasis: LayoutLegendWidth, flexShrink: 0, flexGrow: 0}} />
                    <Box sx={{flex: 1}} display="flex">
                        <Box width="50%">
                            <Typography align='center' sx={{paddingTop: 4}} variant="h6" component="h2">{translate('appPerformance.commandProcessor')}</Typography>
                            <TimeSeriesChart
                                nolegend
                                noframe
                                path={props.path+":statusCodes"} 
                                meta={HttpStatusCodesMeta(app)}
                                units={COUNT}
                                palette={PaletteStatusCodes}
                                height={200}
                                showAll 
                            />
                        </Box>
                        <Box width="50%">
                            <Typography align='center' sx={{paddingTop: 4}} variant="h6" component="h2">{translate('appPerformance.queryProcessor')}</Typography>
                            <TimeSeriesChart
                                nolegend
                                noframe                                
                                path={props.path+":statusCodes"} 
                                meta={HttpStatusCodesMeta(app)}
                                units={COUNT}
                                palette={PaletteStatusCodes}
                                height={200}
                                showAll 
                            />
                        </Box>
                    </Box>
                    
                </Box>

            </MonCard>
            <MonCard caption={translate('appPerformance.executionTime')} >
                <TimeSeriesChart 
                    noframe
                    caption={translate('appPerformance.commandProcessor')}
                    path={props.path+":executionTime:cp"} 
                    meta={CommandProcessorMeta(app)}
                    units={DURATION}
                    height={200}
                    aggs={[]}
                    showAll 
                />
                <TimeSeriesChart 
                    noframe
                    caption={translate('appPerformance.queryProcessor')}
                    height={300}
                    path={props.path+":executionTime:qp"} 
                    meta={QueryProcessorMeta(app)}
                    units={DURATION}
                    aggs={[]}
                    showAll
                />
            </MonCard>
            <Box display={'flex'}>
                <Box width="50%">
                    <AppSlowProjectors height={260} app={app}/>
                </Box>
                <Box width="50%">
                    <AppPartitionsBalance height={260} />
                </Box>
            </Box>
            <MonCard caption={translate('appPerformance.storage')} >
                <TimeSeriesChart 
                    path={props.path+":iops"} 
                    caption={translate('appPerformance.iops')} 
                    meta={StorageIopsMeta(app)}
                    aggs={['avg']} 
                    units={COUNT}
                    height={200}
                    noframe
                    showAll />
                <TimeSeriesChart 
                    path={props.path+":execTime"} 
                    caption={translate('appPerformance.executionTime')} 
                    meta={StorageIopsExecutionTimeMeta(app)}
                    aggs={['avg']} 
                    units={DURATION}
                    height={200}
                    noframe
                    showAll />
                <TimeSeriesChart 
                    path={props.path+":cacheHits"} 
                    caption={translate('appPerformance.cacheHits')} 
                    meta={StorageIopsCacheHitsMeta(app)}
                    aggs={['avg']} 
                    units={PERCENT}
                    height={200}
                    noframe
                    showAll />
            </MonCard>
        </div>
    )
};


export default AppPerformance