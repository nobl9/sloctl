# Get user-category annotations from all projects.
sloctl get annotations -A

# Get both system- and user-category annotations.
sloctl get annotations -A --system --user

# Filter annotations by SLO.
sloctl get annotations --project my-project --slo my-slo

# Require startTime at or after --from and endTime at or before --to.
sloctl get annotations --all-projects \
  --from=2025-03-22T00:00:00Z \
  --to=2025-03-22T23:59:59Z
