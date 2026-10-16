import React from 'react'
import ReactDOM from 'react-dom/client'
import { Route } from 'react-router-dom'
import { createApp } from './src/core'
import './src/index.css'

// A downstream project registering a detail page that has no menu node.
const App = createApp({ extraRoutes: [<Route key="d" path="/users/:id" element={<div>detail</div>} />] })
ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
