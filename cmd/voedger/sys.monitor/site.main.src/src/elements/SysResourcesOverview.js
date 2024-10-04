/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { BooleanField, Button, useTranslate } from 'react-admin';
import CardContent from '@mui/material/CardContent';
import CardActions from '@mui/material/CardActions';
import { Card, Typography, Box } from '@mui/material';
import { DataGrid } from '@mui/x-data-grid';
import MonCard from './MonCard';




export default () => {
    const translate = useTranslate();
    const columns = [
        {
          field: 'node',
          headerName: translate('resourcesOverview.node'),          
          editable: false,
          width: 170,
        },
        {
          field: 'cpu',
          headerName: translate('resourcesOverview.cpu'),
          width: 100,
          editable: false,
          valueFormatter: ({ value }) => `${value}%`
        },
        {
          field: 'memory',
          headerName: translate('resourcesOverview.memory'),
          width: 100,
          editable: false,
          valueFormatter: ({ value }) => `${value}%`
        },
        {
            field: 'totalMemory',
            headerName: translate('resourcesOverview.totalMemory'),
            width: 100,
            editable: false,
        },
        {
            field: 'disk',
            headerName: translate('resourcesOverview.disk'),
            width: 100,
            editable: false,
            valueFormatter: ({ value }) => `${value}%`
        },
          {
              field: 'totalDisk',
              headerName: translate('resourcesOverview.totalDisk'),
              width: 100,
              editable: false,
          },
          {
            field: 'iops',
            headerName: translate('resourcesOverview.iops'),
            width: 100,
            editable: false,
        },
      ];

      const rows = [
        { id: '0', node: 'worker1', cpu: '75', memory: '65', totalMemory: '64G', disk: '25', totalDisk: '100G', iops: '12345' },
        { id: '1', node: 'db1', cpu: '94', memory: '45', totalMemory: '32G', disk: '46', totalDisk: '256G', iops: '42353' },
        { id: '2', node: 'db2', cpu: '55', memory: '92', totalMemory: '32G', disk: '36', totalDisk: '256G', iops: '41153' },
        { id: '3', node: 'db3', cpu: '47', memory: '30', totalMemory: '64G', disk: '88', totalDisk: '256G', iops: '122353' },
      ];  
    
    return (
        <MonCard caption={translate('dashboard.sysResourcesOverview')} toolbar={(<Button sx={{whiteSpace: 'nowrap'}} href="./#/sys-resources" label={translate('common.showDetails')}/>)}>
               <Box sx={{ 
                    width: '100%',
                    '& .warm': {
                        backgroundColor: '#ffcf33',
                      },
                      '& .hot': {
                        backgroundColor: '#ba000d',
                        color: '#fff',
                        fontWeight: 'bold',
                      },
                }}>
                <DataGrid
                    density='compact'
                    rows={rows}
                    columns={columns}
                    rowsPerPageOptions={[5]}
                    disableSelectionOnClick
                    autoHeight={true}
                    hideFooter={true}
                    sx={{
                        boxShadow: 0,
                        border: 0,
                    }}
                    getCellClassName={(params) => {
                        if (params.field === 'cpu' || params.field === 'memory' || params.field === 'disk') {
                            return params.value >= 90 ? 'hot' : (params.value >= 70 ? 'warm' : '');
                        }
                        return '';
                    }}
                />            
                </Box>
        </MonCard>

    )
};
