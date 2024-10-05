/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

import { createSlice } from '@reduxjs/toolkit'

export const ALL = "all"

export const filtersSlice = createSlice({
  name: 'filters',
  initialState: {
    items: {},
  },
  reducers: {
    toggleItem: (state, action) => {
        const target = action.payload.target
        const item = action.payload.item
        const values = action.payload.values

        var current = state.items[target] || [ALL]
        if (item === ALL) {
          state.items[target] = current.includes(ALL)?[]:[ALL]
        } else {
          // item
          if (current.includes(item)) {
            state.items[target] = current.filter((value) => { // remove item
              return value !== item
            })
          } else {

            if (current.includes(ALL)) { // all available but ALL
              current = values                        
              state.items[target] = current.filter((value) => { // remove item
                return value !== item
              })              
            } else {
              state.items[target] = current.concat([item])            
              // disable ALL
              current = state.items[target]
              state.items[target] = current.filter((value) => {
                return value !== ALL
              })  
            }
          }
      }
    },
  },
})

// Action creators are generated for each case reducer function
export const { toggleItem } = filtersSlice.actions

export default filtersSlice.reducer