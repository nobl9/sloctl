# Validate all SLI queries for an existing SLO in the default Project.
sloctl validate sli checkout

# Validate one objective from a manifest over the last 30 minutes.
sloctl validate sli --file ./slo.yaml --slo checkout --objective availability --last 30m

# Validate an explicit time range and return JSON.
sloctl validate sli checkout --from 2026-07-02T10:00:00Z --to 2026-07-02T10:30:00Z --output json
