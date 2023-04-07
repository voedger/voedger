/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { Layout, AppBar, ToggleThemeButton, defaultTheme } from 'react-admin';
import MonMenu from './MonMenu';
import { createTheme, Box, Typography  } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useSelector, useDispatch } from 'react-redux'
import { setInterval } from '../features/filters/appSlice';


const lightTheme = createTheme(defaultTheme, {
    palette: {
    }
});
const darkTheme = createTheme({
    palette: { mode: 'dark' },
});

const MyAppBar = props => {

    const interval = useSelector((state) => state.app.interval)
    const dispatch = useDispatch()

    const handleChange = (event) => {
        dispatch(setInterval(event.target.value))
    };
    
    return (
        <AppBar {...props}>
            <Box flex="1">
                <Typography variant="h6" id="react-admin-title"></Typography>
            </Box>
            <Box>
                    <Select
                        variant='outlined'
                        labelId="demo-simple-select-label"
                        id="demo-simple-select"
                        value={interval}
                        onChange={handleChange}
                        size="small"
                        sx={{
                            marginRight: 2, 
                            minWidth: 120,
                            color: "white",
                            '.MuiOutlinedInput-notchedOutline': {
                              borderColor: 'rgba(255, 255, 255, 0.5)',
                            },
                            '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
                              borderColor: 'rgba(255, 255, 255, 0.5)',
                            },
                            '&:hover .MuiOutlinedInput-notchedOutline': {
                              borderColor: 'rgba(255, 255, 255, 0.5)',
                            },
                            '.MuiSvgIcon-root ': {
                              fill: "white !important",
                            }
                          }}
                    >
                        <MenuItem value={60}>Last 1 Min</MenuItem>
                        <MenuItem value={600}>Last 10 Min</MenuItem>
                        <MenuItem value={1800}>Last 30 Min</MenuItem>
                        <MenuItem value={3600}>Last Hour</MenuItem>
                        <MenuItem value={24 * 3600}>Last 24 Hours</MenuItem>
                    </Select>
            </Box>
            <ToggleThemeButton
                lightTheme={lightTheme}
                darkTheme={darkTheme}
            />
        </AppBar>
    )
};

const MonLayout = (props) => <Layout
    {...props}
    menu={MonMenu}
    appBar={MyAppBar}
/>;

export default MonLayout;