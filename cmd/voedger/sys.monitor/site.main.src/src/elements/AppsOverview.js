/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { Button, useTranslate } from 'react-admin';
import { Typography, Box } from '@mui/material';
import { DataGrid } from '@mui/x-data-grid';
import MonCard from './MonCard';


export default () => {
    const translate = useTranslate();
    const columns = [
        {
          field: 'app',
          headerName: translate('appsOverview.application'),          
          editable: false,
          width: 200,
        },
        {
          field: 'ver',
          headerName: translate('appsOverview.version'),
          width: 100,
          editable: false,
        },
        {
          field: 'partitions',
          headerName: translate('appsOverview.partitions'),
          width: 100,
          editable: false,
        },
        {
          field: 'uptime',
          headerName: translate('appsOverview.uptime'),
          width: 130,
          editable: false,
        },
        {
          field: 'rps',
          headerName: translate('appsOverview.rps'),
          width: 130,
          editable: false,
          renderCell: (params) => {
            return (
              <strong>
                <Button size="small" href={`./#/app-performance?app=${params.row.app.replace("/", ".")}`} label={params.value}/>
              </strong>
            )
          },
        },
      ];

      const rows = [
        { id: '0', app: 'sys/monitor', ver: '0.0.1', 'partitions': '1', uptime: '10d', rps: 12 },
        { id: '1', app: 'sys/registry', ver: '0.1.2', 'partitions': '10', uptime: '10d', rps: 52 },
        { id: '2', app: 'untill/air', ver: '0.2.1', 'partitions': '20', uptime: '4h 10m', rps: 2103 },
      ];  
    
    return (
        <MonCard>
                <Typography variant="h5" component="h2">
                    {translate('dashboard.applicationsOverview')}
               </Typography>

              <Box>
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
                />            
              </Box>
        </MonCard>
    )
};
