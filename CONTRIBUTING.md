Contribution acceptance criteria

1. The change is as small as possible
1. Include proper tests and make all tests pass (unless it contains a test exposing a bug in existing code).
    1. Every new file should have corresponding unit tests, even if the class is exercised at a higher level, such as a feature test.
    1. Unit tests must be 70%+ coverage
1. Your MR initially contains a single commit (please use git rebase -i to squash commits)
1. Your changes can merge without problems (if not please merge master, never rebase commits pushed to the remote server)
1. Does not break any existing functionality
1. Fixes one specific issue or implements one specific feature (do not combine things, send separate merge requests if needed)
1. Migrations (if any) should do only one thing (e.g., either create a table, move data to a new table or remove an old table) to aid retrying on failure (once we implement migrations)
1. Contains functionality we think other users will benefit from too
1. Doesn't add configuration options or settings options since they complicate making and testing future changes (with some exceptions)
1. Changes after submitting the merge request should be in separate commits (no squashing). If necessary, you will be asked to squash when the review is over, before merging.
