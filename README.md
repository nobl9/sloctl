# sloctl

Sloctl is a command-line interface (CLI) for Nobl9.
You can use the sloctl CLI for creating or updating multiple SLOs and
objectives at once as part of CI/CD.

The web user interface is available to give you an easy way to create
and update SLOs and other resources, while sloctl aims to provide a
systematic and/or automated approach to maintaining SLOs as code.

For more details check out
[sloctl user guide](https://docs.nobl9.com/sloctl-user-guide).

## Install

### Prebuilt Binaries

The binaries are available at
[Releases](https://github.com/nobl9/sloctl/releases) page.

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

```shell
docker pull nobl9/sloctl
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
