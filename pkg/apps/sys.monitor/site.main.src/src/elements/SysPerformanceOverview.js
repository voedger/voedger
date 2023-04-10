/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { Button, useTranslate } from 'react-admin';
import {  Box } from '@mui/material';
import MonCard from './MonCard';
import SysPerformanceRps from './SysPerformanceRps';
import SysPerformanceStatusCodes from './SysPerformanceStatusCodes';
import SysPerformanceIops from './SysPerformanceIops';


export default () => {
    const translate = useTranslate();
    
    return (
      <MonCard caption={translate('dashboard.sysPerfOverview')} toolbar={(<Button sx={{whiteSpace: 'nowrap'}} href="./#/sys-performance" label={translate('common.showDetails')}/>)}>
        <Box display="flex">
            <Box width="33%">
              <SysPerformanceRps />
            </Box>
            <Box width="33%">
              <SysPerformanceStatusCodes />
            </Box>
            <Box width="34%">
              <SysPerformanceIops />                
            </Box>
        </Box>
      </MonCard>
    )
};
