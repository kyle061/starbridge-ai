const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { execFileSync } = require('node:child_process');
const root = path.resolve(__dirname, '..');
const context = { module: { exports: {} } };
vm.runInNewContext(fs.readFileSync(path.join(root, 'miniprogram/config.js'), 'utf8'), context, { timeout: 1000 });
const config = context.module.exports;
const projectConfig = require('../project.config.json');
if (!process.env.WX_APP_ID) throw new Error('Set WX_APP_ID to the formal mini program ID');
if (config.demoMode || !config.apiBaseUrl.startsWith('https://')) throw new Error('Configure HTTPS API and set demoMode=false before uploading');
if (projectConfig.appid !== process.env.WX_APP_ID) throw new Error('project.config.json AppID does not match WX_APP_ID');
if (process.env.WX_ACCOUNT_TYPE !== 'formal') throw new Error('Set WX_ACCOUNT_TYPE=formal only after registering a formal mini program.');
const cli = process.env.WX_DEVTOOLS_CLI || '/Applications/wechatwebdevtools.app/Contents/MacOS/cli';
try {
  execFileSync(cli, ['upload', '--project', root, '--version', process.env.RELEASE_VERSION || '1.0.0', '--desc', process.env.RELEASE_DESCRIPTION || '授权视频解析、原片保存、相册保存与历史记录'], { stdio: 'inherit' });
  console.log('Development version uploaded. Submit review and release from the WeChat platform.');
} catch (error) { console.error('Upload failed:', error.message); process.exitCode = 1; }
