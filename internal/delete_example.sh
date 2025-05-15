# Delete the configuration from slo.yaml.
sloctl delete -f ./slo.yaml

# Delete resources from multiple different sources at once.
sloctl delete -f ./slo.yaml -f test/config.yaml -f https://nobl9.com/slo.yaml

# Delete the YAML or JSON passed directly into stdin.
sloctl delete -f - <slo.yaml

# Delete by passing in one or more resource names.
sloctl delete slo my-slo-name

# Delete the configuration from slo.yaml and set project context if it is not defined in file.
sloctl delete -f ./slo.yaml -p slo

# Delete the configurations from all the files located at cwd recursively.
sloctl delete -f '**'

# Delete the configurations from files with 'annotations' name within the whole directory tree.
sloctl delete -f '**/annotations*'
