# Replay SLO 'my-slo' in Project 'my-project' data from 2023-03-02 15:00:00 UTC until now.
sloctl replay -p my-project --from=2023-03-02T15:00:00Z my-slo

# Replay SLOs using file configuration from replay.yaml
sloctl replay -f ./replay.yaml

# Read the configuration from stdin.
sloctl replay <./replay.yaml

# If the project is not set, it is inferred from Nobl9 config.toml for the current context.
# If 'from' is not provided in the config file, you must specify it with '--from' flag.
# Setting 'project' or 'from' via flags does not take precedence over the values set in config.
cat <<EOF | sloctl replay --from=2023-03-02T15:00:00Z
- slo: prometheus-latency
  from: 2023-03-02T16:00:00Z
- slo: datadog-latency
  project: default
- slo: dynatrace-latency
  project: default
  from: 2023-03-02T16:00:00Z
EOF

# Minimal config with project and from set via flags.
cat <<EOF | sloctl replay -p my-project --from 2023-03-02T15:00:00Z
- slo: prometheus-latency
- slo: datadog-latency
EOF
