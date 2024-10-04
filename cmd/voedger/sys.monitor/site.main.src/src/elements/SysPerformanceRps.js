/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { useTranslate } from 'react-admin';
import { Typography, Box } from '@mui/material';
import { ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { PaletteSpringPastels } from '../layout/Palette';
import { DURATION, FormatValue } from '../utils/Units';


const rpsData = [
  { name: 'Commands', value: 4890 },
  { name: 'Queries', value: 8432 },
];

const latencyData = [
  { name: 'Commands', value: 4890123 },
  { name: 'Queries', value: 84322311 },
];


const renderCustomizedRpsLabel = ({ cx, cy, midAngle, innerRadius, outerRadius, percent, index }) => {
  const RADIAN = Math.PI / 180;
  const radius = innerRadius + (outerRadius - innerRadius) * 2;
  const x = cx + radius * Math.cos(-midAngle * RADIAN);
  const y = cy + radius * Math.sin(-midAngle * RADIAN);
  const color = PaletteSpringPastels[index % PaletteSpringPastels.length]

  return (
    <g>
      <text fill={color} x={x} y={y} textAnchor={x > cx ? 'start' : 'end'}  radius={radius}>{`${rpsData[index].name}: ${(percent * 100).toFixed(0)}%`}</text>
      <text fill={color} x={x} y={y+20} textAnchor={x > cx ? 'start' : 'end'}  radius={radius}>{`Avg latency: ${FormatValue(DURATION, latencyData[index].value)}`}</text>
    </g>
    )
};
const SysPerformanceRps = () => {
    const translate = useTranslate();
    
    return (
      <Box>
          <Typography sx={{whiteSpace: 'nowrap'}} variant="h5" align='center'>
            {translate('charts.rps')}
            <span style={{opacity: .5}}> &nbsp;({translate('common.avg')+': 12K'})</span>
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
                  labelLine={false}
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
export default SysPerformanceRps