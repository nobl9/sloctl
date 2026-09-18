# Get all user annotations from all projects.
sloctl get annotations -A

# Get a specific annotation by the annotation name in 'non-default' project.
sloctl get annotation ce1a2a10-d74d-477f-b574-b278ee54e02b -p non-default

# Get all user annotations for 'my-slo' SLO in 'custom' project.
sloctl get annotation -p custom --slo=my-slo

# Get annotations from your default project which mark SLO edits and reviews.
sloctl get annotations --category=SloEdit --category=ReviewNote

# Get all system annotations from all projects.
sloctl get annotations -A --system

# Get all annotations (both system and user) from all projects.
sloctl get annotations -A --system --user

# Filter user annotations by their start and end times.
# `--from` requires `spec.startTime` at or after the specified timestamp.
# `--to` requires `spec.endTime` at or before the specified timestamp.
#
# We're assuming the current date is 2023-03-23T12:00:00Z.
# - Annotations that apply only to yesterday:
sloctl get annotation --from 2023-03-22T00:00:00Z --to 2023-03-22T23:59:59Z -A
# - Annotations that start at or after yesterday (no upper bound):
sloctl get annotation --from 2023-03-22T00:00:00Z -A
# - Annotations that end at or before today (no lower bound):
sloctl get annotation --to 2023-03-23T00:00:00Z -A
