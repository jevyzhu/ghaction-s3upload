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
  const s3path = core.getInput('s3path');
  console.log(`S3 path: ${s3path}!`);
  const binary = chooseBinary()
  const returns  = spawnSync(`${__dirname}/${binary} -p "${file}" -m ${maxcur} -s ${s3path}`, {
    stdio: 'inherit',
    shell: true
  });
} catch (error) {
  core.setFailed(error.message);
}
