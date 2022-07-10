const core = require('@actions/core');
const spawn = require('child_process').spawn

try {
  const file = core.getInput('file');
  console.log(`File pattern: ${file}!`);
  const maxcur = core.getInput('max_cur');
  console.log(`Max tasks numer : ${maxcur}!`);
  const child = spawn(`./s3uploader -p "${file}" -m ${maxcur}`, {
    stdio: 'inherit',
    shell: true
  });
  // child.on('exit', function (code, signal) {
  //   console.log('child process exited with ' +
  //     `code ${code} and signal ${signal}`);
  // });
  // child.stdout.on('data', (data) => {
  //   console.log(`${data}`);
  // });
  // child.stderr.on('data', (data) => {
  //   console.log(`${data}`);
  // });
} catch (error) {
  core.setFailed(error.message);
}
