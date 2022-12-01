# sloctl

The binaries are available at [Releases](https://github.com/nobl9/sloctl/releases) page.

#### Build docker image locally

1. Download Dockerfile and latest linux sloctl binary from the Releases page. Make sure they are in your working directory.
2. Build the image 
```docker build -t <NAME_YOUR_IMAGE> .```
3. Set environment variables if you plan to use them for authenticating with SLOCTL. Reference the [sloctl environment variables Doc](https://docs.nobl9.com/sloctl-user-guide/#configure-sloctl-with-environmental-variables).
4. Run the image 
```
docker run  
-e SLOCTL_CLIENT_ID=$SLOCTL_CLIENT_ID \
-e SLOCTL_CLIENT_SECRET=$SLOCTL_CLIENT_SECRET \
<YOUR_IMAGE_NAME> get slos
``` 
