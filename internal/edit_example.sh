# Edit one SLO from your default Project.
sloctl edit slo my-slo

# Use an alternative editor.
SLOCTL_EDITOR="nano" sloctl edit project default

# Edit one Service from a specific Project.
sloctl edit service my-service -p my-project

# Edit multiple Alert Policies in one session.
sloctl edit alertpolicy policy-a policy-b
