# Get all user annotations from all projects.
sloctl get annotations -A

# Get a specific annotation by the annotation name in 'non-default' project.
sloctl get annotation ce1a2a10-d74d-477f-b574-b278ee54e02b -p non-default

# Get all user annotations for 'my-slo' SLO in 'custom' project.
sloctl get annotation -p custom --slo=my-slo

# Get user annotations from your default project which mark SLO edits and reviews.
sloctl get annotations --category=SloEdit --category=ReviewNote

# Get all system annotations from all projects.
sloctl get annotations -A --system

# Get all annotations (both system and user) from all projects.
sloctl get annotations -A --system --user

# Get user annotations which overlap a specific time window.
# Both `--from` and `--to` flags define a time window which filters overlapping annotations.
# Annotation's time window is defined by `spec.startTime` and `spec.endTime`.
#
# We're assuming the current date is 2023-03-23T12:00:00Z.
# - Annotations that apply only to yesterday:
sloctl get annotation --from 2023-03-22T00:00:00Z --to 2023-03-22T23:59:59Z -A
# - Annotations which time window overlaps a time period starting from yesterday (no upper bound):
sloctl get annotation --from 2023-03-22T00:00:00Z -A
# - Annotations which time window overlaps a time period ending with today (no lower bound):
sloctl get annotation --to 2023-03-23T00:00:00Z -A
