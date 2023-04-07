/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { createSlice } from '@reduxjs/toolkit'

export const ALL = "all"

export const appSlice = createSlice({
  name: 'app',
  initialState: {
    interval: 3600, // seconds
  },
  reducers: {
    setInterval: (state, action) => {
      state.interval = action.payload
    },
  },
})

// Action creators are generated for each case reducer function
export const { setInterval } = appSlice.actions

export default appSlice.reducer