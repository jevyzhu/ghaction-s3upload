const core = require('@actions/core');
const exec = require('child_process').exec;

function os_func() {
    this.execCommand = function(cmd, callback) {
        exec(cmd, (error, stdout, stderr) => {
            if (error) {
                console.error(`exec error: ${error}`);
                return;
            }
            if (stderr) {
                callback(stderr);
            } else {
                callback(stdout);
            }
        });
    }
}


try {
    const file = core.getInput('file');
    console.log(`File pattern: ${file}!`);
    var os = new os_func();
    os.execCommand(`go run . -p ${file}`, function(v) {
        console.log(v)
        core.setOutput("filelist", v);
    });
} catch (error) {
    core.setFailed(error.message);
}