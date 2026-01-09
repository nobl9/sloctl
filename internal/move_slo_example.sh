# Move 'slo-1' and 'slo-2' from your default Project to 'new-project'.
sloctl move slo slo-1 slo-2 --to-project=new-project

# Move 'slo-1' and 'slo-2' from 'old-project' to 'new-project'.
sloctl move slo slo-1 slo-2 -p old-project --to-project=new-project

# Move all SLOs from 'old-project' to 'new-project'.
sloctl move slo -p old-project --to-project=new-project

# Move 'slo-1' and 'slo-2' from 'old-project' to 'new-project' and
# assign 'my-service' in 'new-project' Project for the moved SLOs.
sloctl move slo slo-1 slo-2 -p old-project --to-service=my-service --to-project=new-project

# Move 'slo-1' and 'slo-2' from 'old-project' to 'new-project' and
# detach Alert Policies from both SLOs.
sloctl move slo slo-1 slo-2 -p old-project --detach-alert-policies --to-project=new-project

# Move 'slo-1' and 'slo-2' to 'new-service' within your default Project.
sloctl move slo slo-1 slo-2 --to-service=new-service

# Move 'slo-1' to 'my-service', both the SLO and Service are in 'my-project'.
sloctl move slo slo-1 -p my-project --to-service=my-service

# Move all SLOs in 'my-project' to 'new-service' within the same Project.
sloctl move slo -p my-project --to-service=new-service
