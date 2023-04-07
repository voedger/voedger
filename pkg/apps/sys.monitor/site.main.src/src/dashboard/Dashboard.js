/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import * as React from "react";
import { Title, useTranslate } from 'react-admin';
import ResourcesOverview from "../elements/SysResourcesOverview"
import AppsOverview from "../elements/AppsOverview";
import SysPerformanceOverview from "../elements/SysPerformanceOverview";


export default () => {
    const translate = useTranslate();
    return (
        <div>
            <Title title={translate('dashboard.title')} />
            <ResourcesOverview />
            <SysPerformanceOverview />
            <AppsOverview />
        </div>
    )
};
