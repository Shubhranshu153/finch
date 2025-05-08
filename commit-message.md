The PR title needs to be updated to follow conventional commits format. The current title:

"finch run -h does not produce expected result"

Should be changed to:

"fix: finch run -h does not produce expected result"

This change:
1. Adds the required commit type prefix "fix:" since this is a bug fix
2. Keeps the original description which clearly explains the issue being fixed

This will resolve the pipeline error:
"No release type found in pull request title [...] Add a prefix to indicate what kind of release this pull request corresponds to."

The code changes in the PR correctly fix the issue by:
1. Properly handling "-h" flag for run/create commands to differentiate between help request and hostname setting
2. Maintaining backward compatibility with "--help" flag
3. Preserving the special behavior of "-h" for non-run commands