<!-- markdownlint-disable line-length html -->
<h1 align="center">
   <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://github.com/nobl9/sloctl/assets/48822818/cef721c7-1394-4120-80d1-a5e6eb7c7b7e">
      <source media="(prefers-color-scheme: light)" srcset="https://github.com/nobl9/sloctl/assets/48822818/e137ac37-a299-4a24-951d-197642d31b1a">
      <img alt="N9" src="https://github.com/nobl9/sloctl/assets/48822818/e137ac37-a299-4a24-951d-197642d31b1a" width="500" />
   </picture>
</h1>

<div align="center">
  <table>
    <tr>
      <td>
        <img alt="checks" src="https://github.com/nobl9/sloctl/actions/workflows/checks.yml/badge.svg?event=push">
      </td>
      <td>
        <img alt="tests" src="https://github.com/nobl9/sloctl/actions/workflows/unit-tests.yml/badge.svg?event=push">
      </td>
      <td>
        <img alt="vulnerabilities" src="https://github.com/nobl9/sloctl/actions/workflows/vulns.yml/badge.svg?event=push">
      </td>
    </tr>
  </table>
</div>
<!-- markdownlint-enable line-length html -->

Sloctl is a command-line interface (CLI) for [Nobl9](https://www.nobl9.com/).
You can use sloctl to manage multiple Nobl9 configuration objects
such as [SLOs](https://docs.nobl9.com/getting-started/nobl9-resources/slo),
[Projects](https://docs.nobl9.com/getting-started/nobl9-resources/projects)
or [Alert Policies](https://docs.nobl9.com/getting-started/nobl9-resources/alert-policies).

The web user interface is available to give you an easy way to create
and update SLOs and other resources, while sloctl aims to provide a
systematic and automated approach to maintaining SLOs as code.

You can use it in CI/CD or your terminal power-user workflows :fire:

## Usage

Sloctl includes built-in documentation for each command, to access it, run:

```shell
sloctl <command> --help
```

For more details check out
[sloctl user guide](https://docs.nobl9.com/sloctl-user-guide).

If you want to learn how to fully tame the sloctl's potential, see
[recipes](#recipes) section below.

## Install

### Script

The script requires bash and a minimal set of GNU utilities.
On Windows it will work with either MinGW or Cygwin.
You can either pipe it directly into bash or download the file and run it manually.

```bash
# Using curl:
curl -fsSL https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash

# On systems where curl is not available:
wget -O - -q https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash | bash

# If you prefer to first download the script, inspect it and then run it:
curl -fsSL -o install.bash https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash
# Or with wget:
wget -O install.bash -q https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash
# Once downloaded, set execution permissions:
chmod 700 install.bash
# The script is well documented and comes with additional options.
# You can display the help message by running:
./install.bash --help
```

*NOTE:* If you install sloctl outside of your PATH you must also
change its permissions for the binary to be executable!

```bash
chmod +x <PATH_TO_SLOCTL>
```

### Prebuilt Binaries

The binaries are available at
[Releases](https://github.com/nobl9/sloctl/releases/latest) page.

### Go install

```shell
go install github.com/nobl9/sloctl/cmd/sloctl@latest
```

### Homebrew

```shell
brew tap nobl9/sloctl
brew install sloctl
```

### Docker

Sloctl official images are hosted on [hub.docker.com](https://hub.docker.com/r/nobl9/sloctl).

Here's an example workflow for managing Project object:

1. Export sloctl access keys through environment variables.

   ```shell
   export SLOCTL_CLIENT_ID=<your-client-id>
   export SLOCTL_CLIENT_SECRET=<your-client-secret>
   ```

2. Create a sample Project object and save it to `project.yaml` file.

   ```shell
   cat << EOF > project.yaml
   apiVersion: n9/v1alpha
   kind: Project
   metadata:
     displayName: My Project
     name: my-project
   spec:
     description: Just and example Project :)
   EOF
   ```

3. Apply the project from `project.yaml`.
   To keep STDIN open and allow piping the contents of `project.yaml` into
   the `docker run` command, use interactive mode with `docker run`.

   ```shell
   cat project.yaml | docker run --rm -i \
     -e SLOCTL_CLIENT_ID=${SLOCTL_CLIENT_ID} \
     -e SLOCTL_CLIENT_SECRET=${SLOCTL_CLIENT_SECRET} \
     nobl9/sloctl apply -f -
   ```

4. Fetch the applied Project.

   ```shell
   docker run --rm \
     -e SLOCTL_CLIENT_ID=${SLOCTL_CLIENT_ID} \
     -e SLOCTL_CLIENT_SECRET=${SLOCTL_CLIENT_SECRET} \
     nobl9/sloctl get project my-project
   ```

5. Remove the Project.

   ```shell
   docker run --rm \
     -e SLOCTL_CLIENT_ID=${SLOCTL_CLIENT_ID} \
     -e SLOCTL_CLIENT_SECRET=${SLOCTL_CLIENT_SECRET} \
     nobl9/sloctl delete project my-project
   ```

### Build Docker image locally

1. Download Dockerfile and latest linux sloctl binary from the Releases page.
   Make sure they are in your working directory.
2. Build the image:

   ```shell
   docker build -t <NAME_YOUR_IMAGE> .
   ```

3. Set environment variables if you plan to use them for authenticating with SLOCTL.
   Reference the [sloctl environment variables Doc](https://docs.nobl9.com/sloctl-user-guide/#configure-sloctl-with-environmental-variables).
4. Run the image:

   ```shell
   docker run
   -e SLOCTL_CLIENT_ID=$SLOCTL_CLIENT_ID \
   -e SLOCTL_CLIENT_SECRET=$SLOCTL_CLIENT_SECRET \
   <YOUR_IMAGE_NAME> get slos --no-config-file
   ```

### Recipes

Prerequisites:

- [jq](https://github.com/jqlang/jq), a popular command-line JSON processor.
- [yq](https://github.com/kislyuk/yq) is wrapper over jq,
  it extends jq's capabilities with YAML support.

1. Filter out SLOs with specific integration (_prometheus_ in this example):

```shell
sloctl get slos -A |
  yq -Y \
  --arg source_type "prometheus" \
  '[ .[] | select(
    .spec.objectives[] |
      (.rawMetric and .rawMetric.query[$source_type])
      or
      (.countMetrics and .countMetrics.total[$source_type])
  )]'
```
