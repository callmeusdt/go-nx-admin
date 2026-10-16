import { createContext, useContext } from 'react'
import { DEFAULT_HOME_PATH, type DashboardOptions } from './registry'

export const DashboardOptionsContext = createContext<DashboardOptions | undefined>(undefined)
export const HomePathContext = createContext<string>(DEFAULT_HOME_PATH)

/** The path new sessions land on and the tab bar keeps pinned. */
export const useHomePath = () => useContext(HomePathContext)
