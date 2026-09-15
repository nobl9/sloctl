# Move selected SLOs to another Project.
sloctl move slo api-latency checkout-errors \
  --project old-project \
  --to-project new-project

# Move every SLO from a Project.
sloctl move slo --project old-project --to-project new-project

# Assign an SLO to another Service within the same Project.
sloctl move slo api-latency \
  --project my-project \
  --to-service new-service

# Move an SLO and detach its Alert Policies.
sloctl move slo api-latency \
  --project old-project \
  --to-project new-project \
  --detach-alert-policies
