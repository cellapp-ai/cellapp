import { StrictMode } from 'react'
import { createRoot, hydrateRoot } from 'react-dom/client'
import './index.css'
import './fonts.css'
import './site.css'
import './refinements.css'
import './deploy.css'
import App from './App.tsx'
import { metadata, normalizeRoute } from './content'

const root = document.getElementById('root')!
document.documentElement.dataset.enhanced = 'true'
const route = normalizeRoute(new URLSearchParams(location.search).get('page') || (location.protocol === 'file:' ? '/' : location.pathname))
const meta = metadata(route)
document.title = meta.title
document.querySelector('meta[name="description"]')?.setAttribute('content', meta.description)
const app = <StrictMode><App route={route} /></StrictMode>
if (location.protocol !== 'file:' && root.hasChildNodes() && root.dataset.route === route) hydrateRoot(root, app)
else createRoot(root).render(app)
