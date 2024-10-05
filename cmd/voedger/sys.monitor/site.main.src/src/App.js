/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import * as React from "react";
import { Admin, CustomRoutes, Resource, useTranslate } from 'react-admin';
import { Route } from "react-router-dom";
import jsonServerProvider from 'ra-data-json-server';
import polyglotI18nProvider from 'ra-i18n-polyglot';
import englishMessages from './i18n/en';
import MonLayout from "./layout/MonLayout";
import Dashboard from "./dashboard/Dashboard";
import SysResources from "./elements/SysResources";
import SysPerformance from "./elements/SysPerformance";
import AppPerformance from "./elements/AppPerformance";
import EmuProvider from "./data/EmuProvider";
import { ResMetrics } from "./data/Resources";
import AppPartitionProjectors from "./elements/AppPartitionProjectors";


const i18nProvider = polyglotI18nProvider(locale => {
  if (locale === 'ru') {
      return import('./i18n/ru').then(messages => messages.default);
  }

  // Always fallback on english
  return englishMessages;
}, 'en');


const App = () => {
  const translate = useTranslate();
  return(
    <Admin dataProvider={EmuProvider} layout={MonLayout} i18nProvider={i18nProvider} dashboard={Dashboard} >
        <Resource name={ResMetrics} />
        <CustomRoutes>
              <Route path="/sys-performance"  element={<SysPerformance path="sys-performance" title={translate('menu.sysPerformance')}/>} />
              <Route path="/sys-resources" element={<SysResources path="sys-resources" title={translate('menu.sysResources')} />} />
              <Route path="/app-performance" element={<AppPerformance path="app-performance"/>} />
              <Route path="/app-partition-projectors" element={<AppPartitionProjectors />} />
          </CustomRoutes>
    </Admin>
  )
};

export default App;
