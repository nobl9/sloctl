# Edit one SLO.
sloctl edit slos my-slo --project my-project

# Edit multiple alert policies in one session.
sloctl edit alertpolicies policy-a policy-b --project my-project

# Preview edited changes without persisting them.
sloctl edit services my-service --project my-project --dry-run

# Use another editor for one invocation.
SLOCTL_EDITOR=nano sloctl edit projects my-project
