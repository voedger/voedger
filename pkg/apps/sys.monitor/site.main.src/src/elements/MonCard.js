/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import * as React from "react";
import CardContent from '@mui/material/CardContent';
import { Card, Box, Typography } from '@mui/material';

export default (props) => {

    if (props.noframe) {
        return (
            <Box>
                <Box display="flex">
                    <Box width={'100%'} paddingBottom={1}>
                        <Typography variant="h6" component="h2">
                            {props.caption}
                        </Typography>
                    </Box>
                    <Box>
                        {props.toolbar}
                    </Box>
                </Box>
                {props.children}
            </Box>    
        )
    }

    return (
        <Box sx={{ p: 2.2 }}>
            <Card >
                <CardContent>
                    <Box display="flex">
                        <Box width={'100%'} paddingBottom={1}>
                            <Typography variant="h5" component="h2"  >
                                {props.caption}
                            </Typography>
                        </Box>
                        <Box>
                            {props.toolbar}
                        </Box>
                    </Box>
                    {props.children}
                </CardContent>
            </Card>
        </Box>
    )
};
