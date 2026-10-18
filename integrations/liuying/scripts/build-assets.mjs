import { createRequire } from 'node:module';
import { mkdir, writeFile, copyFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const require = createRequire(import.meta.url);
const lucide = require(process.env.LUCIDE_PACKAGE || 'lucide');
const sharp = require(process.env.SHARP_PACKAGE || 'sharp');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const icons = { 'scan-line': 'ScanLine', link: 'Link', history: 'History', settings: 'Settings', 'clipboard-paste': 'ClipboardPaste', x: 'X', download: 'Download', 'file-down': 'FileDown', 'chevron-right': 'ChevronRight', 'refresh-cw': 'RefreshCw', 'trash-2': 'Trash2', play: 'Play', smartphone: 'Smartphone', film: 'Film', 'arrow-up-right': 'ArrowUpRight', 'hard-drive': 'HardDrive', 'shield-check': 'ShieldCheck' };
await mkdir(path.join(root, 'web/assets'), { recursive: true });
await mkdir(path.join(root, 'miniprogram/assets'), { recursive: true });
for (const [name, key] of Object.entries(icons)) for (const [suffix, color] of [['', '#7c8480'], ['-active', '#197b56'], ['-white', '#ffffff']]) {
  const nodes = lucide[key].map(([tag, attributes]) => `<${tag} ${Object.entries(attributes).map(([k, v]) => `${k}="${v}"`).join(' ')}/>`).join('');
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">${nodes}</svg>`;
  await writeFile(path.join(root, 'web/assets', name + suffix + '.svg'), svg);
  await sharp(Buffer.from(svg)).resize(72, 72).png().toFile(path.join(root, 'miniprogram/assets', name + suffix + '.png'));
}
const packagePath = require.resolve((process.env.LUCIDE_PACKAGE || 'lucide') + '/package.json');
await copyFile(path.join(path.dirname(packagePath), 'LICENSE'), path.join(root, 'web/assets/LUCIDE-LICENSE'));
console.log('Generated Lucide web and mini program assets');
