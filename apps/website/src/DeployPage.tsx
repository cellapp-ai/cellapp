import React, { useRef, useState } from 'react';
import { deploymentPrompt } from './content';

export default function DeployPage() {
  const [copyStatus, setCopyStatus] = useState('');
  const prompt = useRef<HTMLParagraphElement>(null);

  async function copyPrompt() {
    try {
      if (navigator.clipboard && window.isSecureContext) await navigator.clipboard.writeText(deploymentPrompt);
      else {
        const input = document.createElement('textarea');
        input.value = deploymentPrompt;
        input.style.position = 'fixed';
        input.style.opacity = '0';
        document.body.append(input);
        input.select();
        const copied = document.execCommand('copy');
        input.remove();
        if (!copied) throw new Error('Copy unavailable');
      }
      setCopyStatus('Copied. Paste this into the AI tool working on your project.');
    } catch {
      if (prompt.current) {
        const range = document.createRange();
        range.selectNodeContents(prompt.current);
        window.getSelection()?.removeAllRanges();
        window.getSelection()?.addRange(range);
      }
      setCopyStatus('Copy unavailable. The prompt is selected for manual copying.');
    }
  }

  return <React.Fragment>
    <section className="wrap deploy-hero">
      <div className="eyebrow">Build · deploy · share</div>
      <h1>From idea to<br /><em>a live app.</em></h1>
      <p>Cellapp hosts your static frontend as a shareable HTTPS app with access controls. Give the coding Agent working on your project one request; it builds locally, you approve access in your browser, and it returns the published URL.</p>
    </section>

    <section className="wrap deploy-prompt" aria-labelledby="deploy-prompt-title">
      <div className="deploy-prompt-heading"><div><div className="section-label">One request</div><h2 id="deploy-prompt-title">Start in your project.</h2></div><button className="button primary" onClick={() => { void copyPrompt(); }}>Copy deployment prompt</button></div>
      <p ref={prompt}>{deploymentPrompt}</p>
      <p className="deploy-copy-status" role="status">{copyStatus}</p>
    </section>

    <section className="wrap deploy-flow" aria-labelledby="deploy-flow-title">
      <div className="section-label">How it works</div>
      <h2 id="deploy-flow-title">Three steps to a link.</h2>
      <ol>
        <li><span>01</span><h3>Ask your AI</h3><p>It reads the project, confirms a static build and prepares the publishing tools.</p></li>
        <li><span>02</span><h3>Approve in your browser</h3><p>On first use, open the verification link and compare the code. Keep passwords and credentials out of the chat.</p></li>
        <li><span>03</span><h3>Get the URL</h3><p>The build and upload run locally. A successful publication returns a stable HTTPS address for your app.</p></li>
      </ol>
    </section>

    <section className="wrap deploy-details" aria-labelledby="deploy-fit-title">
      <div><div className="section-label">Current capabilities</div><h2 id="deploy-fit-title">What can I deploy?</h2><p>Cellapp hosts static frontends, including plain HTML and apps that build to static files. Projects needing a continuously running server require a real static alternative before deployment.</p></div>
      <div><div className="section-label">Access and data</div><h2>What happens next?</h2><p>Visitors sign in to Cellapp; restricted apps need an additional grant. Controlled application tables can store shared data. Paid app sales are a future direction and are not available in this flow.</p></div>
    </section>

    <section className="wrap deploy-next"><p>For agents: <a href="https://cellapp.ai/deploy/AGENTS.md">read the full deployment instructions</a>. Manage your apps in the <a href="/dashboard">dashboard</a>.</p></section>
  </React.Fragment>;
}
