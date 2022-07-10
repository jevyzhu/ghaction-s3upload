const core = require('@actions/core');
const spawnSync = require('child_process').spawnSync

try {
  const file = core.getInput('file');
  console.log(`File pattern: ${file}!`);
  const maxcur = core.getInput('max_cur');
  console.log(`Max tasks numer : ${maxcur}!`);
  const returns  = spawnSync(`${__dirname}/s3uploader -p "${file}" -m ${maxcur}`, {
    stdio: 'inherit',
    shell: true
  });
} catch (error) {
  core.setFailed(error.message);
}
