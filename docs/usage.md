---
layout: docs
---

{::options toc_levels="2" /}

# Artifactory Secrets Engine

The Artifactory secrets engine dynamically generates access tokens base on the
user and/or the groups configured in roles.
This allows short lived access tokens to be created and avoids the need to
distribute credentials or API keys to applications and CI/CD systems.

* Table of Contents
{:toc}

## Setup

Most secrets engines must be configured in advance before they can perform
their functions. These steps are usually completed by an operator or
configuration management tool.

 1. Install the plugin in the [`plugin_directory`](https://www.vaultproject.io/docs/configuration/index.html#plugin_directory):

    ```
    vault write sys/plugins/catalog/artifactory \
        sha_256="$(shasum -a 256 /path/to/plugin-directory/vault-plugin-secrets-artifactory | cut -d' ' -f1)" \
        command="vault-plugin-secrets-artifactory"
    ```

 1. Enable the Artifactory secrets engine:

    ```
    $ vault secrets enable --plugin-name=artifactory -path=artifactory plugin
    Success! Enabled the artifactory secrets engine at: artifactory/
    ```

 1. Configure the engine with either user/password or API key credentials:

    ```
    $ vault write artifactory/config \
        address=https://example.com/artifactory/ \
        api_key=<API KEY>
    ```

    or:

    ```
    $ vault write artifactory/config \
        address=https://example.com/artifactory/ \
        username=<USERNAME> \
        password=<PASSWORD>
    ```

 1. Configure a role:

    ```
    $ vault write artifactory/roles/reader \
        member_of_groups=readers
    ```

 1. Issue an access token:

    ```
    $ vault read --format=json artifactory/token/privileged
    {
        "data": {
            "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
            "scope": "api:* member-of-groups:readers",
            "token_type": "Bearer"
        }
    }
    ```

 1. Use the access token to interact with artifactory:

    ```
    curl -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...' \
         https://example.com/artifactory/api/system/ping
    ```

## Considerations

### Token Scope, Expiry and Revocation

Due to the restrictions Artifactory places on Access Token creation,
administrators should take into consideration which level of Artifactory access
the engine is configured with.

If administrator scope credentials are supplied:

 * Admin-scope access tokens can be created
 * [Transient/Non-existing users][non-existing-users] can be created
 * Access tokens can have any expiry/TTL

See [Generating Admin Tokens][generating-admin-tokens] in the Artifactory documentation for more details.

If user scoped credentials are supplied:

 * Access tokens are limited to the same or a subset of privileges of the issuing user
 * Access tokens will be tied to the same user
 * Access tokens can have an expiry/TTL of less than or equal to the globally configured maximum in the Artifactory instance (`artifactory.access.token.non.admin.max.expires`).

Artifactory instances are configured with a global setting: `minimum-revocable-expiry`.
This dictates that any token whose expiry is shorter than this settings **cannot be revoked** and must naturally expire instead.
Only tokens with an expiry larger than this value can be actively revoked.

The Artifactory secrets engine will always attempt to revoke access tokens when
the secret lease expires, however if revocation fails the *engine assumes that
the token is irrevocable and will not retry*.

See [Generating Expirable Tokens][generating-expirable-tokens] in the Artifactory documentation for more details.

## API

The Artifactory secrets engine has a full HTTP API.
Please see the [Artifactory secrets engine API]({{ site.baseurl }}/api) for more details.

[generating-expirable-tokens]: https://www.jfrog.com/confluence/display/ACC/Access+Tokens#AccessTokens-GeneratingExpirableTokens
[generating-admin-tokens]: https://www.jfrog.com/confluence/display/ACC/Access+Tokens#AccessTokens-GeneratingAdminTokens
[non-existing-users]: https://www.jfrog.com/confluence/display/ACC/Access+Tokens#AccessTokens-SupportAuthenticationforNon-ExistingUsers
