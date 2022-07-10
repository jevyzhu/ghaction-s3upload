const core = require('@actions/core');
const spawnSync = require('child_process').spawnSync

function chooseBinary() {
    if (process.platform === 'linux' && process.arch === 'x64') {
        return "main-linux-amd64"
    }
}

try {
  const file = core.getInput('file');
  console.log(`File pattern: ${file}!`);
  const maxcur = core.getInput('max_cur');
  console.log(`Max tasks numer : ${maxcur}!`);
  const binary = chooseBinary()
  const returns  = spawnSync(`${__dirname}/${binary} -p "${file}" -m ${maxcur}`, {
    stdio: 'inherit',
    shell: true
  });
} catch (error) {
  core.setFailed(error.message);
}
