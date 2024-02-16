<!-- markdownlint-disable line-length html -->
<h1 align="center">
   <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://user-images.githubusercontent.com/32738712/185149468-dc07f5d9-68c0-4922-a006-7baf6a08eaac.png">
      <source media="(prefers-color-scheme: light)" srcset="https://user-images.githubusercontent.com/32738712/185148352-bea80385-c772-4842-8f7b-6838bb08a3f4.png">
      <img alt="N9" src="https://user-images.githubusercontent.com/32738712/185148352-bea80385-c772-4842-8f7b-6838bb08a3f4.png" width="500" />
   </picture><br/>
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
