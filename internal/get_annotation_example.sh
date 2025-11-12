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

# Get user annotations which apply to a specific time range.
# We're assuming the current date is 2023-03-23T12:00:00Z:
# - Annotations that apply only to yesterday:
sloctl get annotation --from 2023-03-22T00:00:00Z --to 2023-03-22T23:59:59Z -A
# - Annotations that have 'spec.startTime' AFTER yesterday:
sloctl get annotation --from 2023-03-22T00:00:00Z -A
# - Annotations that have 'spec.endTime' BEFORE today:
sloctl get annotation --to 2023-03-23T00:00:00Z -A
