# Delete resources described in a file.
sloctl delete --file ./resources.yaml

# Preview deletion without persisting changes.
sloctl delete --file ./resources.yaml --dry-run

# Read definitions from standard input.
sloctl delete --file - <./resources.yaml

# Delete resources described by supported files recursively.
sloctl delete --file '**'

# Delete resources by name.
sloctl delete slos availability latency --project my-project
