[![CircleCI](https://circleci.com/gh/jsok/vault-plugin-secrets-artifactory.svg?style=svg)](https://circleci.com/gh/jsok/vault-plugin-secrets-artifactory)

# Vault Plugin: Artifactory Database Backend

This is a backend plugin to be used with [Hashicorp Vault](https://www.github.com/hashicorp/vault).
It uses the [Artifactory REST API to dynamically issue Access Tokens](https://www.jfrog.com/confluence/display/ACC/Access+Tokens#AccessTokens-RESTAPI).

## Work in Progress

This plugin is still in development! An initial release is being tracked under the [v0.1.0 - Initial release Milestone](https://github.com/jsok/vault-plugin-secrets-artifactory/milestone/1).

## Documentation

See the [documentation](https://jsok.github.io/vault-plugin-secrets-artifactory/)

To build the documentation locally:

```
cd docs
bundle install
bundle exec jekyll serve
```
