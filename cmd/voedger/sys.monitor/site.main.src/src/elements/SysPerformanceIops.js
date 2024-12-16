/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { useTranslate } from 'react-admin';
import { Typography, Box } from '@mui/material';
import { ResponsiveContainer, PieChart, Pie, Cell} from 'recharts';
import { PaletteSpringPastels } from '../layout/Palette';


const rpsData = [
  { name: 'Get', value: 1234 },
  { name: 'GetBatch', value: 5432 },
  { name: 'Put', value: 200 },
  { name: 'PutBatch', value: 905 },
  { name: 'Read', value: 863 },
];


const renderCustomizedRpsLabel = ({ cx, cy, midAngle, innerRadius, outerRadius, percent, index }) => {
  return `${rpsData[index].name}: ${(percent * 100).toFixed(0)}%`
};

const SysPerformanceIops = () => {
    const translate = useTranslate();
    
    return (
      <Box>
          <Typography sx={{whiteSpace: 'nowrap'}} variant="h5" align='center'>
            {translate('charts.iops')}
            <span style={{opacity: .5}}> &nbsp;({translate('common.avg')+': 17K'})</span>
          </Typography>                

          <Box width="100%">
            <ResponsiveContainer width="100%" aspect={2.5}>
              <PieChart width={200} height={200}>
                <Pie
                  dataKey="value"
                  isAnimationActive={false}
                  data={rpsData}
                  cx="50%"
                  cy="50%"
                  outerRadius={60}
                  innerRadius={30}
                  fill="#8884d8"
                  labelLine={true}
                  label={renderCustomizedRpsLabel}
                >
                {rpsData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={PaletteSpringPastels[index % PaletteSpringPastels.length]} />
                ))}                   
                </Pie>
              </PieChart>
            </ResponsiveContainer>
          </Box>
        </Box>            

    )
};


export default SysPerformanceIops