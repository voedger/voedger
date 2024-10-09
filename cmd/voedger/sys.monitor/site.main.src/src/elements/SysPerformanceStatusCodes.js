/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { useTranslate } from 'react-admin';
import { Typography, Box } from '@mui/material';
import { ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { PaletteStatusCodes } from '../layout/Palette';


const scData = [
  { name: '2xx', value: 12452 },
  { name: '4xx', value: 320 },
  { name: '5xx', value: 1234 },
];


const renderCustomizedScLabel = ({ cx, cy, midAngle, innerRadius, outerRadius, percent, index }) => {
  return `${scData[index].name}: ${(percent * 100).toFixed(0)}%`
};

const SysPerformanceStatusCodes = () => {
    const translate = useTranslate();
    
    return (
      <Box>
          <Typography sx={{whiteSpace: 'nowrap'}} variant="h5" align='center'>
            {translate('charts.statusCodes')}            
          </Typography>                
          <ResponsiveContainer width="100%" aspect={2.5}>
            <PieChart width={200} height={200}>
              <Pie
                dataKey="value"
                isAnimationActive={false}
                data={scData}
                cx="50%"
                cy="50%"
                outerRadius={60}
                innerRadius={30}
                fill="#8884d8"
                labelLine={true}
                label={renderCustomizedScLabel}
              >
              {scData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={PaletteStatusCodes[index % PaletteStatusCodes.length]} />
              ))}                   
              </Pie>
            </PieChart>
          </ResponsiveContainer>
      </Box>               

    )
};

export default SysPerformanceStatusCodes