# Fluent pipeline tests

## Local run

To run tests you need:

* Unix/Linux shell (VM with Linux, WSL)
* docker

Integration tests for all 3 agent deployments can be run using the script `run-tests.sh`.
This script has 1 input parameter with name of test scenario:

* `fluentd`
* `fluentbit`
* `fluentbit-ha`

### Problems during local run

#### Tests failed due the `\r` at the end

This issue can occurs due work on `Windows`, but run tests in `WSL2`, for example in `Ubuntu 24.04`.
Also, most probably you configured in `Git` work with line ends in `Windows-style`, but commit `in Unix-style`.

You can manually convert line end in any IDE/editors (VSC, Notepad++, etc.).

Or use the CLI tool:

```bash
apt install dos2unix
find fluent-pipeline-test/ -type f -exec dos2unix {} \;
```
