# Terraform Providers

## About

This directory is synchronized between [openshift/terraform-providers][providersrepo] and a [git subtree][gitsubtree] in [openshift/installer][providerssubtree].
We use the separate terraform-providers repo in order to create container images for use in the build system, but also keep the provider dependencies in the
Installer repo to allow local builds in a disconnected environment. Furthermore, the providers in the Installer repo are the canonical versions that will be used
by the Installer. In the case of a discrepency between the synchronized repos, the Installer will build and use the provider from the in-tree code.

## Working with git subtree

### Prerequisites

Git subtree commands are available as part of [git/contrib][contrib]. You will probably need to install subtree, which
you can do on Fedora by:

```shell
dnf install git-subtree
```

### Suggested Workflow

Whenever changes are made to the providers, the changes need to be committed to both repos: changes to the Installer repo will enable the updated functionality;
changes to the Terraform-Providers repo will ensure that the build process can use pre-built binaries.

Changes can be made and tested directly in your local Installer repo, then pushed to both remotes.

### Pushing from Installer to Terraform Providers

You can make changes and test directly in the Installer repo subtree then push those changes 

### Pulling from Terraform Providers to Installer

In order to update the Installer gitsubtree with changes from the Terraform Providers repo

[providersrepo]: github.com/openshift/terraform-providers
[providerssubtree]: https://github.com/openshift/installer/tree/master/terraform/providers
[gitsubtree]: https://github.com/git/git/blob/master/contrib/subtree/git-subtree.txt
[contrib]: https://github.com/git/git/tree/master/contrib