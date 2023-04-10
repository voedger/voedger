/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { configureStore } from '@reduxjs/toolkit'
import filtersReducer from './features/filters/filtersSlice'
import appReducer from './features/filters/appSlice'

export default configureStore({
  reducer: {
    filters: filtersReducer,
    app: appReducer,
  },
})