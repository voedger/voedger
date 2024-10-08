/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { Bar, BarChart, CartesianGrid, Cell, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useTranslate } from 'react-admin';
import { useSearchParams } from "react-router-dom";
import MonCard from './MonCard';



const AppPartitionProjectors = (props) => {

    let [searchParams] = useSearchParams();    
    const app = searchParams.get("app")
    const partition = searchParams.get("partition")
    const translate = useTranslate();

    const data = [
      {name: 'sys.Collection', p: 0},
      {name: 'air.Dashboard', p: 21},
      {name: 'air.OrderDates', p: -20},
      {name: 'air.PBillDates', p: -5},
      {name: 'air.UpdateSubscrpition', p: 0},
      {name: 'air.TransactionHistory', p: -32},
      {name: 'air.TablesOverview', p: 0},
      {name: 'air.NewRestaurantVatProjector', p: -18},
      {name: 'sys.WLogDates', p: 0},
      {name: 'sys.SendEmailVerificationCodeProjector', p:0},
    ]
    
    return (
        <MonCard caption={translate('appPerformance.projectorsProgressAtPartition')+": "+partition}>
            <ResponsiveContainer width="100%" height={400}>
              <BarChart
                width={500}
                height={300}
                data={data}
                layout="vertical"
                margin={{
                  top: 5,
                  right: 30,
                  left: 300,
                  bottom: 5,
                }}
              >
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" />
                <YAxis dataKey="name" type="category" />
                <Tooltip />
                <ReferenceLine y={0} stroke="#000" />
                <Bar name='Overrun' dataKey="p" fill="#333" >
                  {data.map((entry, index) => (
                    <Cell cursor="pointer" fill={data[index].p > 0 ? "#bdcf32" : "#ea5545"} key={`cell-${index}`} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
        </MonCard>

    )

};


export default AppPartitionProjectors