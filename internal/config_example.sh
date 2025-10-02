# Run interactive form which adds a new context to your config.toml file.
sloctl config add-context

# Fetch the current context name.
sloctl config current-context

# Display detailed information about the current context in YAML format.
sloctl config current-context --verbose

# List all context names.
sloctl config get-contexts

# Display detailed information about every context in TOML format.
sloctl config get-contexts --verbose --output=TOML

# Fetch the current user ID.
sloctl config current-user

# Display detailed information about the current user in YAML format.
sloctl config current-user --verbose

# Display interactive selection of contexts to use as default.
sloctl config use-context

# Use "my-context" as a default context.
sloctl config use-context my-context

# Display interactive form which lets you select the old context name and type in the new one.
sloctl config rename-context

# Rename "old-ctx" to "new-ctx".
sloctl config rename-context old-ctx new-ctx

# Display interactive selection of context to delete (multiple choice).
sloctl config delete-context

# Delete "context-1" and "context-2" from your configuration file.
sloctl config delete-context contetx-1 context-2
