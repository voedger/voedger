/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import englishMessages from 'ra-language-english';

const customEnglishMessages = {
    ...englishMessages,
    menu: {
        performance: 'Performance',
        resources: 'Resources',
        sysPerformance: 'System Performance',
        sysResources: 'System Resources',
        appPerformance: 'App Performance',
    },
    dashboard: {
        title: 'Dashboard',
        sysPerfOverview: 'System Performance Overview',
        sysResourcesOverview: 'System Resources Overview',
        applicationsOverview: 'Applications Overview',
    },
    charts: {
        cpuUsage: 'CPU Usage',
        memUsage: 'Memory Usage',
        diskUsage: 'Disk Usage',
        diskIO: 'Disk I/O',
        rps: 'RPS',
        iops: 'IOPS',
        statusCodes: 'Status Codes',
    },
    resourcesOverview: {
        node: 'Node',
        cpu: 'CPU',
        memory: 'Memory',
        totalMemory: 'Total Mem',
        disk: 'Disk',
        totalDisk: 'totalDisk',
        iops: 'IOPS',
    },
    appsOverview: {
        application: 'Application',
        version: 'Version',
        partitions: 'Partitions',
        uptime: 'Uptime',
        rps: 'RPS',
    },
    performanceOverview: {
        total503: 'Total 503',
    },
    appPerformance: {
        commandProcessor: 'Command Processor',
        queryProcessor: 'Query Processor',
        executionTime: 'Execution Time',
        cacheHits: 'Cache Hits',
        projectorsProgress: 'Projectors Overrun',
        projectorsProgressAtPartition: 'Projectors Overrun at Partition',
        partitionsBalance: 'Partitions Balance',
        projector: 'Projector',
        partition: 'App Partition',
        lag: 'Lag',
        storage: 'Storage',
        iops: 'IOPS',
        overall: 'Overall',
    },
    sysPerformance: {
        executionTime: 'Execution Time',
        cacheHits: 'Cache Hits',
        top5ByRt: 'Top 5 by request time',
        top5ByRps: 'Top 5 by RPS',
        bottom5ByCacheHits: 'Bottom 5 by cache hits',
        top5ByBatchSize: 'Top 5 by batch size',
        worstApps: 'Worst Apps'
    },
    common: {
        showDetails: 'Show Details',
        average: 'Average',
        showAll: 'Show all',
        avg: 'Avg',
    }
};

export default customEnglishMessages;