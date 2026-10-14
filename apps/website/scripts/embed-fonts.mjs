import { readFile, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);
const families = [['space-grotesk', 'Space Grotesk', [400, 500, 600, 700]], ['dm-mono', 'DM Mono', [400, 500]]];
const rules = [];
for (const [id, name, weights] of families) {
  for (const weight of weights) {
    const bytes = await readFile(require.resolve(`@fontsource/${id}/files/${id}-latin-${weight}-normal.woff2`));
    rules.push(`@font-face{font-family:'${name}';font-style:normal;font-weight:${weight};font-display:swap;src:url(data:font/woff2;base64,${bytes.toString('base64')}) format('woff2')}`);
  }
}
await writeFile(new URL('../src/fonts.css', import.meta.url), rules.join('\n'));
console.log('Embedded 6 local font weights');
