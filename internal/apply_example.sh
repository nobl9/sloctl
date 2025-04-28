# Apply the configuration from slo.yaml.
sloctl apply -f ./slo.yaml

# Apply resources from multiple different sources at once.
sloctl apply -f ./slo.yaml -f test/config.yaml -f https://nobl9.com/slo.yaml

# Apply the YAML or JSON passed directly into stdin.
sloctl apply -f - <slo.yaml

# Apply the configuration from slo.yaml and set project if it is not defined in file.
sloctl apply -f ./slo.yaml -p slo

# Apply the configurations from all the files located at cwd recursively.
sloctl apply -f '**'

# Apply the configurations from files with 'annotations' name within the whole directory tree.
sloctl apply -f '**/annotations*'

# Apply the SLO(s) from slo.yaml and import its/their data from 2023-03-02T15:00:00Z until now.
sloctl apply -f ./slo.yaml --replay --from=2023-03-02T15:00:00Z
