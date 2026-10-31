// Writes the licences of every package in the bundle next to it.
const fs = require('fs');
const path = require('path');

const out = path.join(__dirname, '../../awsassetinventory/web/vendor/graph.LICENSES.txt');
const lock = require('./package-lock.json');
const sections = [];

for (const [dir, info] of Object.entries(lock.packages)) {
  if (!dir || info.dev) continue;
  const pkgDir = path.join(__dirname, dir);
  const name = dir.replace(/^.*node_modules\//, '');
  const file = fs.readdirSync(pkgDir).find((f) => /^(licen[cs]e|copying)/i.test(f));
  const text = file ? fs.readFileSync(path.join(pkgDir, file), 'utf8').trim() : `License: ${info.license}`;
  sections.push(`${name}@${info.version}\n${'='.repeat(name.length + info.version.length + 1)}\n\n${text}\n`);
}

sections.sort();
fs.writeFileSync(out, sections.join('\n\n'));
console.log(`Wrote ${sections.length} licences to ${path.relative(process.cwd(), out)}`);
