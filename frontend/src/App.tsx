import { createApp } from './core'

const App = createApp({ sessionMode: import.meta.env.VITE_NX_SESSION_MODE === 'cookie' ? 'cookie' : 'bearer' })

export default App
