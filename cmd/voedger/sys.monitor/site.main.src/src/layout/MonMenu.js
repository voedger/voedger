/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import * as React from 'react';
import { useState } from 'react';
import Box from '@mui/material/Box';
import PerformanceIcon from '@mui/icons-material/QueryStats';
import ResourcesIcon from '@mui/icons-material/Memory';
import AppsIcon from '@mui/icons-material/Apps';

import {
    useTranslate,
    DashboardMenuItem,
    MenuItemLink,
    useSidebarState,
} from 'react-admin';

import SubMenu from './SubMenu';

const MonMenu = (props) => {
    const [state, setState] = useState({
        menuUntillAir: false,
        menuSysMonitor: false,
        menuSysRegistry: false,
    });
    const translate = useTranslate();
    const [open] = useSidebarState();
    const { dense } = props;

    const handleToggle = (menu) => {
        setState(state => ({ ...state, [menu]: !state[menu] }));
    };

    return (
        <Box
            sx={{
                width: open ? 250 : 50,
                marginTop: 1,
                marginBottom: 1,
                transition: theme =>
                    theme.transitions.create('width', {
                        easing: theme.transitions.easing.sharp,
                        duration: theme.transitions.duration.leavingScreen,
                    }),
            }}
        >
            <DashboardMenuItem />
            <MenuItemLink
                to="/sys-performance"
                state={{ _scrollToTop: true }}
                primaryText={translate(`menu.sysPerformance`, {
                    smart_count: 2,
                })}
                leftIcon={<PerformanceIcon/>}
                dense={dense}
            />
            <MenuItemLink
                to="/sys-resources"
                state={{ _scrollToTop: true }}
                primaryText={translate(`menu.sysResources`, {
                    smart_count: 2,
                })}
                leftIcon={<ResourcesIcon/>}
                dense={dense}
            />          
            <SubMenu
                handleToggle={() => handleToggle('menuUntillAir')}
                isOpen={state.menuUntillAir}
                name="untill/air"
                translate={false}
                icon={<AppsIcon />}
                dense={dense}
            >
                <MenuItemLink
                    to="/app-performance?app=untill.air"
                    state={{ _scrollToTop: true }}
                    primaryText={translate(`menu.performance`, {
                        smart_count: 2,
                    })}
                    leftIcon={<PerformanceIcon/>}
                    dense={dense}
                />
            </SubMenu>
            <SubMenu
                handleToggle={() => handleToggle('menuSysMonitor')}
                isOpen={state.menuSysMonitor}
                translate={false}
                name="sys/monitor"
                icon={<AppsIcon />}
                dense={dense}
            >
                <MenuItemLink
                    to="/app-performance?app=sys.monitor"
                    state={{ _scrollToTop: true }}
                    primaryText={translate(`menu.performance`, {
                        smart_count: 2,
                    })}
                    leftIcon={<PerformanceIcon/>}
                    dense={dense}
                />
            </SubMenu>
            <SubMenu
                handleToggle={() => handleToggle('menuSysRegistry')}
                isOpen={state.menuSysRegistry}
                name="sys/registry"
                translate={false}
                icon={<AppsIcon />}
                dense={dense}
            >
                <MenuItemLink
                    to="/app-performance?app=sys.registry"
                    state={{ _scrollToTop: true }}
                    primaryText={translate(`menu.performance`, {
                        smart_count: 2,
                    })}
                    leftIcon={<PerformanceIcon/>}
                    dense={dense}
                />
            </SubMenu>
        </Box>
    );
};

export default MonMenu;