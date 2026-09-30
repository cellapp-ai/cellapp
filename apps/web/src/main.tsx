import { createRoot } from 'react-dom/client';
import { StrictMode } from 'react';
import '@fontsource/space-grotesk/latin-400.css';
import '@fontsource/space-grotesk/latin-500.css';
import '@fontsource/space-grotesk/latin-600.css';
import '@fontsource/space-grotesk/latin-700.css';
import '@fontsource/dm-mono/latin-400.css';
import '@fontsource/dm-mono/latin-500.css';
import { App } from '@/App';
import '@/styles/web.css';

const root = document.getElementById('root');
if (!root) throw new Error('Web root element is missing');
createRoot(root).render(<StrictMode><App /></StrictMode>);
