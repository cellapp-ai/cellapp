import { useEffect, useRef, useState } from 'react';
import * as React from 'react';
import type { ReactNode } from 'react';
import { articles, articleBySlug, docGuides, github, plans } from './content';
import type { Article, Cadence, DocLocale } from './content';
import { Button } from './components/ui/button';
import { wordmark } from './logo';
import DeployPage from './DeployPage';

function Link({ href, children, className = '', lang }: { href: string; children: ReactNode; className?: string; lang?: string }) {
  const localFile = typeof location !== 'undefined' && location.protocol === 'file:';
  const [withoutHash, hash] = href.split('#');
  const [pathname, query] = withoutHash.split('?');
  const destination = localFile && pathname === '/dashboard' ? `https://cellapp.ai${href}` : localFile && href.startsWith('/') ? `?page=${encodeURIComponent(pathname)}${query ? `&${query}` : ''}${hash ? `#${hash}` : ''}` : href;
  return <a href={destination} className={className} lang={lang} {...(href.startsWith('https:') ? { target: '_blank', rel: 'noreferrer' } : {})}>{children}</a>;
}
function Brand() { return <Link href="/" className="brand"><img className="brand-wordmark" src={wordmark} width="493" height="100" alt="Cellapp home" /></Link>; }
function Header({ route }: { route: string }) {
  const [open, setOpen] = useState(false);
  return <header className="wrap site-header"><nav aria-label="Primary navigation"><Brand /><button className="menu-button" aria-expanded={open} aria-controls="site-navigation" onClick={() => setOpen(!open)}>{open ? 'Close' : 'Menu'}</button><div id="site-navigation" className={`nav-links ${open ? 'is-open' : ''}`}>{[['/', 'Home'], ['/deploy/', 'Deploy'], ['/pricing/', 'Pricing'], ['/docs/', 'Docs'], ['/blog/', 'Blog']].map(([href, label]) => <a key={href} href={typeof location !== 'undefined' && location.protocol === 'file:' ? `?page=${encodeURIComponent(href)}` : href} className={(href === '/' ? route === '/' : route.startsWith(href)) ? 'active' : ''} aria-current={(href === '/' ? route === '/' : route.startsWith(href)) ? 'page' : undefined}>{label}</a>)}<Link className="nav-cta" href="/dashboard">Dashboard ↗</Link></div></nav></header>;
}
function Footer() { return <footer className="wrap"><Brand /><div><Link href="/deploy/">Deploy</Link><Link href="/pricing/">Pricing</Link><Link href="/docs/">Documentation</Link><Link href="/blog/">Journal</Link><Link href={github}>GitHub ↗</Link></div><span>© 2026 Cellapp</span></footer>; }
function PageHero({ label, title, copy }: { label: string; title: string; copy: string }) { return <section className="page-hero wrap"><div className="eyebrow blue">{label}</div><h1>{title}</h1><p>{copy}</p></section>; }
function CodeBlock({ code, label = 'Terminal', locale = 'en' }: { code: string; label?: string; locale?: DocLocale }) {
  const [status, setStatus] = useState('');
  const block = useRef<HTMLElement>(null);
  const copied = locale === 'en' ? 'Copied to clipboard' : '已复制到剪贴板';
  async function copy() {
    try {
      if (navigator.clipboard && window.isSecureContext) await navigator.clipboard.writeText(code);
      else { const input = document.createElement('textarea'); input.value = code; input.style.position = 'fixed'; input.style.opacity = '0'; document.body.append(input); input.select(); const ok = document.execCommand('copy'); input.remove(); if (!ok) throw new Error('copy unavailable'); }
      setStatus(copied);
    } catch { if (block.current) { const range = document.createRange(); range.selectNodeContents(block.current); window.getSelection()?.removeAllRanges(); window.getSelection()?.addRange(range); } setStatus(locale === 'en' ? 'Copy unavailable. Command selected for manual copying.' : '无法自动复制，已选中命令，请手动复制。'); }
  }
  return <div className="code-block"><div className="code-heading"><span>{label}</span><Button variant="outline" size="sm" className="copy-button" onClick={copy} aria-label={locale === 'en' ? `Copy ${label.toLowerCase()} command` : '复制部署提示词'}>{status === copied ? (locale === 'en' ? 'Copied' : '已复制') : (locale === 'en' ? 'Copy' : '复制')}</Button></div><pre tabIndex={0} aria-label={label}><code ref={block}>{code}</code></pre><div className="copy-status" role="status">{status}</div></div>;
}
function Home() { return <><div className="wrap hero"><div><div className="eyebrow">Frontend shipping, simplified</div><h1>your apps,<br /><em>your rules.</em></h1><p className="hero-copy">Cellapp hosts your static frontend as a shareable HTTPS app with access controls.</p><p className="hero-use">Tell the coding Agent working on your project to follow the Cellapp deployment guide, approve access in your browser, then share the app URL it returns.</p><div className="actions"><Link className="button primary" href="/deploy/">Copy Agent prompt →</Link><Link className="button ghost" href="/docs/">Read the guide →</Link></div></div><div className="terminal" aria-label="Illustrative deployment preview"><div className="terminal-top"><span className="dot" /><span className="dot" /><span className="dot" /><span>Cellapp — publishing preview</span></div><div className="terminal-body"><div><span className="prompt">→</span> Your AI publishes the finished build</div><div className="comment">uploading your static build...</div><div><span className="prompt">✓</span> your app is live</div><div className="app-card"><div className="app-name"><span className="app-icon">A</span>acme-dashboard</div><span className="live">LIVE</span></div></div></div></div><div className="wrap ticker"><strong>Static by design</strong><span>No backend to run</span><span>HTTPS by default</span><span>Share when ready</span></div><section className="wrap"><div className="section-label">The 4S vision</div><h2>Shape, Ship, Share and Sell.</h2><div className="four-s"><div><h3>Shape</h3><p>Make the app with the coding Agent in your project.</p></div><div><h3>Ship</h3><p>Build locally and publish the static result.</p></div><div><h3>Share</h3><p>Give intended viewers access to your app link.</p></div><div><h3>Sell <span>Coming later</span></h3><p>Paid app sales are a future direction.</p></div></div><div className="section-label">A dependable handoff</div><h2>Your frontend deserves a real home.</h2><div className="features">{[['One request to ship', 'Your AI publishes the build you already have. No backend setup or infrastructure work.'], ['A link that stays put', 'Give your team a stable HTTPS URL they can bookmark, review, and come back to.'], ['Access on your terms', 'Share the application with the right people and keep deployment credentials separate.']].map(([title, body]) => <article className="feature" key={title}><h3>{title}</h3><p>{body}</p></article>)}</div></section><section className="flow"><div className="wrap flow-grid"><div><div className="section-label">From local to live</div><h2>One request.<br />Big handoff.</h2><p className="flow-intro">Build with the tools you like, then hand over the finished static files.</p></div><div className="steps">{[['Build as usual', 'Use Vite, Astro, or any tool that outputs static files.'], ['Publish the folder', 'Your AI picks the build command and output directory, then publishes with Cellapp'], ['Share the link', 'Open the returned HTTPS URL, verify the app, then send it to your reviewer.']].map(([title, text], i) => <div className="step" key={title}><span className="step-number">0{i + 1}</span><div><h3>{title}</h3><p>{text}</p></div></div>)}</div></div></section><section className="wrap"><div className="section-label">Keep moving</div><h2>Everything around the deploy.</h2><div className="resource-grid">{[['/pricing/', 'Find your fit', 'Compare plans for a first demo or a growing team.', 'Explore pricing'], ['/deploy/', 'Make your first link', 'One request for your coding agent, from setup to publication.', 'Open deployment guide'], ['/blog/', 'Read the field notes', 'Small lessons in frontend shipping and review.', 'Visit the journal']].map(([href, title, text, label]) => <article key={href}><h3>{title}</h3><p>{text}</p><Link href={href}>{label} →</Link></article>)}</div></section><section className="wrap final"><h2>Make the next demo feel like a real product.</h2><p>Ship something people can open.</p><Link href="/deploy/" className="button primary">Get started →</Link></section></>; }
function Pricing() {
  const [cadence, setCadence] = useState<Cadence>('monthly');
  return <><PageHero label="Pricing" title="A plan for the way you ship." copy="Start with your next project. Give your team more room as the work grows." /><section className="wrap"><div className="callout pricing-notice"><strong>Plan preview.</strong> Names, prices, limits, and billing terms are proposed and not available for purchase yet.</div><div className="pricing-toggle"><div className="toggle" role="group" aria-label="Billing cadence">{(['monthly', 'yearly'] as const).map(c => <button key={c} aria-pressed={c === cadence} onClick={() => setCadence(c)}>{c === 'monthly' ? 'Monthly' : 'Yearly'}</button>)}</div><span className="save" role="status">{cadence === 'yearly' ? 'Monthly equivalent · billed annually' : 'Monthly billing preview'}</span></div><div className="plans">{plans.map(plan => <article key={plan.id} className={`plan ${plan.recommended ? 'recommended' : ''}`}><div className="plan-tag">{plan.recommended ? 'Recommended' : 'Plan preview'}</div><h2>{plan.name}</h2><p className="desc muted">{plan.audience}</p><div className="price">${plan[cadence]}<small>{plan.monthly === 0 ? ' / forever' : ' / month'}</small></div><p className="billing-detail">{plan.monthly === 0 ? 'Free plan preview' : cadence === 'yearly' ? `$${plan.yearly * 12} per year · save ${Math.round((1 - plan.yearly / plan.monthly) * 100)}%` : `$${plan.monthly} per month`}</p><ul><li>{plan.apps}</li><li>{plan.access}</li><li>{plan.support} support</li><li>HTTPS included</li></ul><Link className={`button ${plan.recommended ? 'primary' : 'ghost'}`} href={`/docs/?plan=${plan.id}&cadence=${cadence}`}>Explore {plan.name} →</Link></article>)}</div><div className="comparison"><h2>Compare the essentials.</h2><div className="table-scroll"><table><caption className="sr-only">Proposed plan feature comparison</caption><thead><tr><th scope="col">Capability</th>{plans.map(p => <th scope="col" key={p.id}>{p.name}</th>)}</tr></thead><tbody>{[['Active apps', 'apps'], ['Access', 'access'], ['Support', 'support']].map(([label, key]) => <tr key={label}><th scope="row">{label}</th>{plans.map(p => <td key={p.id}>{p[key as 'apps' | 'access' | 'support']}</td>)}</tr>)}<tr><th scope="row">HTTPS</th>{plans.map(p => <td key={p.id}>Included</td>)}</tr></tbody></table></div></div><div className="pricing-faq"><h3>Can I purchase a plan here?</h3><p>Not yet. These plans are a preview. The next step opens the quickstart with your selection; no checkout or subscription is created.</p><h3>What should I deploy?</h3><p>A finished static build containing index.html and its assets. Start with the documentation if your project is new to static hosting.</p><Link className="button ghost" href="/docs/">Read the quickstart →</Link></div></section></>;
}
function Docs({ locale }: { locale: DocLocale }) {
  const guide = docGuides[locale];
  const otherLocale = locale === 'en' ? 'zh-CN' : 'en';
  const otherRoute = otherLocale === 'en' ? '/docs/' : '/docs/zh-CN/';
  const [active, setActive] = useState('getting-started');
  const [selection, setSelection] = useState('');
  useEffect(() => {
    const query = new URLSearchParams(location.search);
    const plan = plans.find(p => p.id === query.get('plan'));
    if (plan) setSelection(`${plan.name} plan preview · ${query.get('cadence') === 'yearly' ? 'yearly' : 'monthly'} billing selected. No subscription has been created.`);
    if (location.hash && guide.sections.some(section => `#${section.id}` === location.hash)) setActive(location.hash.slice(1));
    const observer = new IntersectionObserver(entries => {
      const visible = entries.filter(entry => entry.isIntersecting);
      if (visible.length) setActive(visible[0].target.id);
    }, { rootMargin: '-10% 0px -60% 0px' });
    document.querySelectorAll('[data-doc-section]').forEach(element => observer.observe(element));
    return () => observer.disconnect();
  }, [guide]);
  return <>
    <section className="page-hero wrap docs-hero">
      <div className="eyebrow blue">{guide.label}</div>
      <h1>{guide.title}</h1>
      <p>{guide.summary}</p>
      <nav className="language-switch" aria-label="Documentation language">
        <Link href={`/docs/#${active}`} className={locale === 'en' ? 'active' : ''} lang="en">English</Link>
        <Link href={`/docs/zh-CN/#${active}`} className={locale === 'zh-CN' ? 'active' : ''} lang="zh-CN">中文</Link>
      </nav>
    </section>
    <section className="wrap content-grid docs-grid">
      <aside className="side-nav" aria-label={locale === 'en' ? 'Documentation sections' : '文档章节'}>
        <div className="side-title">{guide.outline}</div>
        {guide.sections.map(section => <a href={`#${section.id}`} key={section.id} className={active === section.id ? 'active' : ''} aria-current={active === section.id ? 'location' : undefined} onClick={() => setActive(section.id)}>{section.title}</a>)}
      </aside>
      <div className="docs-body">
        {selection ? <div className="callout" role="status">{selection}</div> : null}
        {guide.sections.map((section, index) => <article id={section.id} data-doc-section className="page-section" key={section.id}>
          <div className="docs-section-top"><div className="section-label">{section.title}</div><Link className="docs-translate" href={`${otherRoute}#${section.id}`} lang={otherLocale}>{locale === 'en' ? '阅读中文' : 'Read in English'} ↗</Link></div>
          <h2>{section.heading}</h2>
          <p className="docs-intro">{section.intro}</p>
          {section.steps.map(step => <section className="docs-step" key={step.heading}>
            <h3>{step.heading}</h3><p>{step.text}</p>
            {step.points ? <ul>{step.points.map(point => <li key={point}>{point}</li>)}</ul> : null}
            {step.note ? <aside className="callout">{step.note}</aside> : null}
            {step.code ? <CodeBlock code={step.code} label={locale === 'en' ? 'Agent prompt' : 'Agent 提示词'} locale={locale} /> : null}
            {step.link ? <Link className="text-link" href={step.link.href}>{step.link.label} ↗</Link> : null}
          </section>)}
          {guide.sections[index + 1] ? <a className="next-doc" href={`#${guide.sections[index + 1].id}`}>{guide.next}: {guide.sections[index + 1].title} →</a> : <Link href="/deploy/" className="button primary">{guide.finish} →</Link>}
        </article>)}
      </div>
    </section>
  </>;
}
const formatDate = (date: string) => new Intl.DateTimeFormat('en-US', { dateStyle: 'medium', timeZone: 'UTC' }).format(new Date(date));
function ArticleMeta({ article }: { article: Article }) { return <div className="meta"><span>{article.category}</span><time dateTime={article.date}>{formatDate(article.date)}</time><span>{article.readMinutes} min read</span></div>; }
function Blog() { return <><PageHero label="Journal" title="The handoff is part of the product." copy="Notes on static shipping, frontend craft, and the small decisions that make a link worth sharing." /><section className="wrap"><div className="article-list">{articles.map((a, i) => <Link key={a.slug} className={`article-card ${i === 0 ? 'featured' : ''}`} href={`/blog/${a.slug}/`}><ArticleMeta article={a} /><h2>{a.title}</h2><p>{a.summary}</p><span className="article-action">Read article →</span></Link>)}</div></section></>; }
function ArticlePage({ article }: { article: Article }) { return <section className="wrap article-page"><article className="article-body"><Link href="/blog/" className="text-link">← Journal</Link><ArticleMeta article={article} /><h1>{article.title}</h1><p className="article-lede">{article.summary}</p>{article.body.map(block => <section key={block.heading} className="article-section"><h2>{block.heading}</h2>{block.paragraphs.map(p => <p key={p}>{p}</p>)}{block.code ? <CodeBlock code={block.code} /> : null}</section>)}<div className="article-next"><Link href="/blog/">← All field notes</Link><Link href="/docs/" className="button primary">Read the quickstart →</Link></div></article></section>; }
export default function App({ route }: { route: string }) {
  const article = articleBySlug(route.split('/')[2]);
  const page = route === '/' ? <Home /> : route === '/deploy/' ? <DeployPage /> : route === '/pricing/' ? <Pricing /> : route === '/docs/' ? <Docs locale="en" /> : route === '/docs/zh-CN/' ? <Docs locale="zh-CN" /> : route === '/blog/' ? <Blog /> : article ? <ArticlePage article={article} /> : <PageHero label="404" title="That page is not here." copy="Use the navigation to find pricing, documentation, or the journal." />;
  return <React.Fragment><a className="skip" href="#main">Skip to content</a><Header route={route} /><main id="main">{page}</main><Footer /></React.Fragment>;
}
