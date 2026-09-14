# Convert a file.
sloctl convert openslo --file ./service.yaml

# Convert standard input.
sloctl convert openslo --file - <./service.yaml

# Convert OpenSLO files recursively from the current directory.
sloctl convert openslo --file '**'

# Convert and apply in one pipeline.
sloctl convert openslo --file ./service.yaml | sloctl apply --file -
