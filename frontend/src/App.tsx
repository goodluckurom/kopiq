import { useEffect, useState } from 'react'
import { Library } from './components/library/Library'
import { QuickPaste } from './components/quickpaste/QuickPaste'

function routeFromHash(): 'library' | 'quick-paste' {
  const hash = window.location.hash.replace(/^#/, '') || '/'
  if (hash === '/quick-paste' || hash.startsWith('/quick-paste')) {
    return 'quick-paste'
  }
  return 'library'
}

export default function App() {
  const [route, setRoute] = useState(routeFromHash)

  useEffect(() => {
    const onHash = () => setRoute(routeFromHash())
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  if (route === 'quick-paste') {
    return <QuickPaste />
  }

  return <Library />
}
