import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './styles/index.css'
import { initAppearance } from './lib/appearance'
import { applyPlatformAttr } from './lib/platform'

applyPlatformAttr()
initAppearance()

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
