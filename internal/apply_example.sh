# Apply definitions from a file.
sloctl apply --file ./resources.yaml

# Preview an apply without persisting changes.
sloctl apply --file ./resources.yaml --dry-run

# Apply definitions from multiple sources.
sloctl apply --file ./project.yaml --file ./slos.yaml

# Apply supported files recursively.
sloctl apply --file '**'

# Read definitions from standard input.
sloctl apply --file - <./slo.yaml

# Apply SLOs and import historical data.
sloctl apply --file ./slo.yaml --replay --from=2025-03-02T15:00:00Z
