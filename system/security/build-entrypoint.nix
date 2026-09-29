# Prevent accidental deployment that skips the wrapper's preparation checks.
# Root and users who can edit the source can change this policy.
{
  buildContext ? null,
  ...
}:
{
  assertions = [
    {
      assertion =
        buildContext == {
          schemaVersion = 1;
          entrypoint = "gjallarctl";
        };
      message = "Direct GjallarOS flake builds are disabled. Use rebuild (or gjallarctl installer deploy); these commands prepare and validate a staged source.";
    }
  ];
}
