/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { useTranslate } from 'react-admin';
import { useNavigate } from "react-router-dom";
import { Bar, BarChart, CartesianGrid, Cell, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import MonCard from './MonCard';


export default (props) => {
    const translate = useTranslate();
    const data = [
      {name: 'P1', p: 0},
      {name: 'P2', p: 213},
      {name: 'P3', p: -20},
      {name: 'P4', p: -5},
      {name: 'P5', p: 0},
      {name: 'P6', p: -327},
      {name: 'P7', p: 0},
      {name: 'P8', p: -180},
      {name: 'P9', p: 0},
      {name: 'P10', p:0},
    ]

    const navigate = useNavigate();

    const handleClick = (data, index) => {
      navigate(`/app-partition-projectors?app=${props.app}&partition=${index+1}`);
    };
    
    return (
        <MonCard caption={translate('appPerformance.projectorsProgress')}>
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
                <ReferenceLine y={0} stroke="#000" />
                <Bar name='Overrun' dataKey="p" fill="#333" onClick={handleClick} >
                  {data.map((entry, index) => (
                    <Cell cursor="pointer" fill={data[index].p > 0 ? "#bdcf32" : "#ea5545"} key={`cell-${index}`} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
        </MonCard>

    )
};
