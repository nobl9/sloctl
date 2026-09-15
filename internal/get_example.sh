# Get every SLO in the configured default project.
sloctl get slos

# Get selected SLOs as JSON.
sloctl get slos availability latency --project my-project --output json

# Read resource names from standard input.
printf '%s\n' availability latency | sloctl get slos --project my-project
