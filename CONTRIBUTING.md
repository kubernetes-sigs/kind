# Cloud Contributing Guidelines

This guide serves to set clear expectations for everyone involved with the project so that we can improve it together. Following these guidelines will help to ensure a positive experience for every maintainer.

## General workflow

Here's the general workflow to follow for every change to the code in this repository:

- A new functionality is proposed and accepted by the team, then the corresponding Jira ticket is created.
- During the team's sprint planning, the team will add the required detail to the Jira ticket.
- A developer then creates a Pull Request with the labels _wip_ + _dont merge_.
- When the functionality is implemented, the developer will then change the PR labels to _ok-to-review_ (deleting the previous ones).
- Another developer (peer) reviews the PR, testing it minimally and ensuring the code meets the standards (such as correctness, readability, maintainability and consistency).
- Once the review is done, the peer will change the PR labels to _ok-to-test_.
- At this point, the QA engineer will do the testing and evaluate if the PR adds the funcionality described and doesn't compromise any other.
- When the PR is ready to be merge, the QA engineer set the label _ok-to-merge_.
- The developer then can merge the PR at anytime. In case the funcionality has several PRs, the developer will merge them in the correct order.

If the PR requires changes in the documentation or tests, the developer should open them separately. 

## Special labels

### Releasing and PRs classification

The following labels should be used to ease the releasing process:

- `<release>`: A label for each release is created in order to manage which PR is included (e.g. `0.8.1`).
- `documentation`: Indicates that the PR requires the documentation team to review it.
- `cherry-pick`: When the functionality must be present in existing branches, this label must be used in those PRs (they must have the same name as the original PR).
- `need-cherry-pick-<branch>`: Applied to flag that a PR needs to be cherry-picked to a specific branch (e.g. `need-cherry-pick-master`, `need-cherry-pick-branch-0.17.0-0.6`). These labels are branch-specific and new ones are created per release.
- `bugfix`: For PRs that fix bugs.
- `feature`: For PRs that implement new functionality.
- `improvement`: For PRs that enhance existing functionality without adding new features.
- `blocked`: Marks a PR that is blocked waiting on an external dependency or decision.

### CICD flow

Jenkins has its own labels to control which pipeline stages are executed:

**Build and security:**
- `skip doUT`: Skip unit tests.
- `skip doGrypeScan`: Skip the Grype security vulnerability scan.

**Documentation checks** (applied when the `DOC` stage is not relevant for the PR):
- `skip doStratioDocsChecks`: Skip all Stratio documentation checks.
- `skip deadExternalLinks`: Skip the dead external links check.
- `skip doEmptyLines`: Skip the empty lines check.
- `skip doLanguageCheck`: Skip the language/spelling check.
- `skip doOrphanAssets`: Skip the orphan assets check.

### Testing

AT-<provider/flavour>-smoke: These labels execute smoke tests on the specific cloud provider/flavour as part of the PR validations (e.g. "AT-eks-smoke" for EKS).

## Getting Started

To get started with this project, please read the documentation first.

- Official Documentation (published): http://antora.labs.stratio.com/es/cloud-provisioner/0.4/introduction.html
- Official Documentation (unpublished): [stratio-docs](stratio-docs/en/modules/ROOT/pages/quick-start-guide.adoc)

## Contact Information

The maintainer's team can be reached on this [email](clouds-integration@stratio.com), but preffer the Stratio internal channels.

