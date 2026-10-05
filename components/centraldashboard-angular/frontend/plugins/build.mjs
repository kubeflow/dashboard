import { build } from 'esbuild';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const pluginsDirectory = dirname(fileURLToPath(import.meta.url));
const assetDirectory = resolve(pluginsDirectory, '..', 'src', 'assets', 'plugins');

const plugins = [
  { id: 'hello-lit', entry: 'hello-lit/src/hello-lit.ts' },
  { id: 'hello-react', entry: 'hello-react/src/hello-react.ts' },
];

await Promise.all(plugins.map(plugin => build({
  bundle: true,
  entryPoints: [resolve(pluginsDirectory, plugin.entry)],
  format: 'iife',
  legalComments: 'none',
  minify: true,
  outfile: resolve(assetDirectory, plugin.id, 'main.js'),
  target: ['es2017'],
})));
