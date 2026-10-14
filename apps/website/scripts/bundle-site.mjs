import { createReadStream, createWriteStream } from 'node:fs';
import { pipeline } from 'node:stream/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
const require = createRequire(import.meta.url);
const inline = require('html-inline');
const root = path.resolve(import.meta.dirname, '..');
await pipeline(
  createReadStream(path.join(root, '.build/parcel/index.html')),
  inline({ basedir: path.join(root, '.build/parcel') }),
  createWriteStream(path.join(root, '.build/bundle.html')),
);
console.log('Bundled inline HTML into .build/bundle.html');
