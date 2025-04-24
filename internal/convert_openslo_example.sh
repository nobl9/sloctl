# Convert the OpenSLO definitions from service.yaml.
cat <<EOF > ./service.yaml
apiVersion: openslo/v1
kind: Service
metadata:
  annotations:
    nobl9.com/metadata.project: my-project
  name: example-service
spec:
  description: Example service description
EOF
sloctl convert openslo -f ./service.yaml

# Convert definitions from multiple different sources at once.
sloctl convert openslo -f ./service.yaml -f test/config.yaml

# Convert the YAML or JSON passed directly into stdin.
sloctl convert openslo -f - <service.yaml

# Convert definitions from all the files located at cwd recursively.
sloctl convert openslo -f '**'

# Convert definitions from files with 'data-sources' name within the whole directory tree.
sloctl convert openslo -f '**/data-sources*'

# Apply converted definitions in one step.
sloctl convert openslo -f ./service.yaml | sloctl apply -f -
