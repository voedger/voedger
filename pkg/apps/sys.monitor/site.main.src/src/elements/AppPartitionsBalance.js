/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { useTranslate } from 'react-admin';
import { Box } from '@mui/material';
import { DataGrid } from '@mui/x-data-grid';
import MonCard from './MonCard';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';




export default (props) => {
    const translate = useTranslate();
    const data = [
      {name: 'P1', q: 1200, c: 300},
      {name: 'P2', q: 1110, c: 240},
      {name: 'P3', q: 843, c: 176},
      {name: 'P4', q: 921, c: 342},
      {name: 'P5', q: 1301, c: 321},
      {name: 'P6', q: 1190, c: 201},
      {name: 'P7', q: 541, c: 21},
      {name: 'P8', q: 5430, c: 105},
      {name: 'P9', q: 20, c: 5},
      {name: 'P10', q: 650, c: 190},
    ]
    return (
        <MonCard caption={translate('appPerformance.partitionsBalance')}>
            <ResponsiveContainer width="100%" height={props.height}>
              <BarChart
                width={500}
                height={300}
                data={data}
                margin={{
                  top: 5,
                  right: 30,
                  left: 20,
                  bottom: 5,
                }}
              >
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Bar name='Commands' dataKey="c" stackId="a" fill="#fd7f6f" />
                <Bar name='Queries' dataKey="q" stackId="a" fill="#7eb0d5" />
              </BarChart>
            </ResponsiveContainer>
        </MonCard>

    )
};
