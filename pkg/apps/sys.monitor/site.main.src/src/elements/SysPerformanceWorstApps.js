/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { Error, Title, useDataProvider, useTheme } from 'react-admin';
import { Bars } from 'svg-loaders-react'
import { Typography, Box } from '@mui/material';
import CardContent from '@mui/material/CardContent';
import { Card, CardHeader } from '@mui/material';
import { useTranslate } from 'react-admin';
import { COUNT, DURATION, FormatValue, PERCENT } from '../utils/Units';
import { useSelector } from 'react-redux';
import { PaletteSpringPastels } from '../charts/TimeSeriesChart';
import { ResWorstApps } from '../data/Resources';
import { useEffect, useState } from 'react';

export default (props) => {

    const [theme] = useTheme();
    const dataProvider = useDataProvider()

    const appInterval = useSelector((state) => state.app.interval)

    const [loading, setLoading] = useState(true);
    const [error, setError] = useState();
    const [worstApps, setWorstApps] = useState(); 
    const [interval, setInterval] = useState(appInterval);

    const renderMoreSection = function(data, caption, unit) {
        return (
            <Box>
                <table cellpadding="4">
                    <thead>
                        <tr>
                            <th colspan="2" style={{borderBottomColor: theme.palette.divider, borderBottomWidth: 1, borderBottomStyle: 'solid'}}><Typography sx={{fontWeight: 'bolder'}} >{caption}</Typography></th>
                        </tr>
                    </thead>
                    <tbody>
                        {data.map((entry, index) => (
                            <tr key={`cell-${index}`}>
                                <td><Typography>{entry.app}</Typography></td>
                                <td><Typography>{FormatValue(unit, entry.value)}</Typography></td>
                            </tr>
                        ))}                   
                    </tbody>            
                </table>
            </Box>
        )
    }
    const reload = (interval) => {
        dataProvider.getOne(ResWorstApps, {meta: {interval: interval}})
            .then(({ data }) => {
                setWorstApps(data);
                setLoading(false);
            })
            .catch(error => {
                setError(error);
                setLoading(false);
            })
    }

    const Section = (props) => (
        <Card variant='outlined'>
            <CardHeader 
                title={props.title} 
                sx={{backgroundColor: theme.palette.action.selected, padding: '.5rem 1.3rem'}} 
                titleTypographyProps = {{
                    variant: "h6",
                    component: "h2"
                }}
            />
            <CardContent>
                <Box display={"flex"} gap={'2rem'}>
                    {props.children}
                </Box>
            </CardContent>
        </Card>
    )

    useEffect(() => {
        reload(interval)
    }, []);
    
    if (appInterval != interval) { // Interval has changed
        setInterval(appInterval)
        setLoading(true)
        setError(false)
        reload(appInterval);
    }

    const translate = useTranslate();
    
    if (error) {
        return (
            <Error />
        )
    }

    if (loading) {
        return (
            <Box width="100%" height={props.height} sx={{ border: '1px solid #ccc' }} display='flex' alignItems={'center'} justifyContent={'center'}>
                <Bars stroke={PaletteSpringPastels[1]} fill={PaletteSpringPastels[1]} width="60"/>
            </Box>
        )
    }

    return (
        <Box>
            <Typography variant="h6" component="h2" sx={{margin: '1rem 0'}}>
                {translate('sysPerformance.worstApps')}
            </Typography>
            <Box display="flex" flexWrap={'wrap'} gap={'2rem'}>
                <Section title='Get'>
                    {renderMoreSection(worstApps.moreGetTop5AppsByRT, translate('sysPerformance.top5ByRt'), DURATION)}
                    {renderMoreSection(worstApps.moreGetTop5AppsByRPS, translate('sysPerformance.top5ByRps'), COUNT)}
                    {renderMoreSection(worstApps.moreGetBottom5AppsByCacheHits, translate('sysPerformance.bottom5ByCacheHits'), PERCENT)}
                </Section>
                <Section title='GetBatch'>
                    {renderMoreSection(worstApps.moreGetBatchTop5AppsByRT, translate('sysPerformance.top5ByRt'), DURATION)}
                    {renderMoreSection(worstApps.moreGetBatchTop5AppsByRPS, translate('sysPerformance.top5ByRps'), COUNT)}
                    {renderMoreSection(worstApps.moreGetBatchBottom5AppsByCacheHits, translate('sysPerformance.bottom5ByCacheHits'), PERCENT)}
                </Section>
                <Section title='Read'>
                    {renderMoreSection(worstApps.moreReadTop5AppsByRT, translate('sysPerformance.top5ByRt'), DURATION)}
                    {renderMoreSection(worstApps.moreReadTop5AppsByRPS, translate('sysPerformance.top5ByRps'), COUNT)}
                </Section>
                <Section title='Put'>
                    {renderMoreSection(worstApps.morePutTop5AppsByRT, translate('sysPerformance.top5ByRt'), DURATION)}
                    {renderMoreSection(worstApps.morePutTop5AppsByRPS, translate('sysPerformance.top5ByRps'), COUNT)}
                </Section>
                <Section title='PutBatch'>
                    {renderMoreSection(worstApps.morePutBatchTop5AppsByRT, translate('sysPerformance.top5ByRt'), DURATION)}
                    {renderMoreSection(worstApps.morePutBatchTop5AppsByRPS, translate('sysPerformance.top5ByRps'), COUNT)}
                    {renderMoreSection(worstApps.morePutBatchTop5AppsByBatchSize, translate('sysPerformance.top5ByBatchSize'), COUNT)}
                </Section>
            </Box>
        </Box>
    )

}