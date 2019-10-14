---
layout: "docs"
---

{::options toc_levels="2" /}
# Artifactory Database Secrets Engine

Artifactory is one of the supported plugins for the database secrets engine. This
plugin generates database credentials dynamically based on configured roles for
Artifactory.

See the [database secrets engine][database-docs] docs for
more information about setting up the database secrets engine.

## Getting Started

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

## Setup

1. Enable the database secrets engine if it is not already enabled:

    ```text
    $ vault secrets enable database
    Success! Enabled the database secrets engine at: database/
    ```

    By default, the secrets engine will enable at the name of the engine. To
    enable the secrets engine at a different path, use the `-path` argument.

1. Configure Vault with the proper plugin and connection information:

    ```text
    $ vault write database/config/my-artifactory-database \
        plugin_name="artifactory-database-plugin" \
        allowed_roles="internally-defined-role,externally-defined-role" \
        username=vault \
        password=myPa55word \
        address=http://localhost:8081/artifactory
    ```

    Or using an API key

    ```text
    $ vault write database/config/my-artifactory-database \
        plugin_name="artifactory-database-plugin" \
        allowed_roles="art-reader" \
        api_key=api-key \
        address=http://localhost:8081/artifactory
    ```

1. Configure a role that maps a name in Vault to a role definition in Artifactory.

    ```text
    $ vault write database/roles/my-role \
        db_name="my-artifactory-database" \
        creation_statements='{"artifactory_groups": ["art-reader"] \
        default_ttl="1h" \
        max_ttl="24h"
     ```

## Usage

After the secrets engine is configured and a user/machine has a Vault token with
the proper permission, it can generate credentials.

1. Generate a new credential by reading from the `/creds` endpoint with the name
of the role:

    ```text
    $ vault read database/creds/art-reader
    Key                Value
    ---                -----
    lease_id           database/creds/art-reader/2f6a614c-4aa2-7b19-24b9-ad944a8d4de6
    lease_duration     1h
    lease_renewable    true
    password           8cab931c-d62e-a73d-60d3-5ee85139cd66
    username           v-root-e2978cd0-
    ```

The returned token can be used with the username for basic auth or by itself for token authentication.

## API

The full list of configurable options can be seen in the [Artifactory database
plugin API]({{ site.baseurl }}/api) page.

For more information on the database secrets engine's HTTP API please see the
[Database secrets engine API][database-api] page.


[database-docs]: https://www.vaultproject.io/docs/secrets/databases/index.html
[database-api]: https://www.vaultproject.io/api/secret/databases/index.html
[generating-expirable-tokens]: https://www.jfrog.com/confluence/display/ACC/Access+Tokens#AccessTokens-GeneratingExpirableTokens
[generating-admin-tokens]: https://www.jfrog.com/confluence/display/ACC/Access+Tokens#AccessTokens-GeneratingAdminTokens
[non-existing-users]: https://www.jfrog.com/confluence/display/ACC/Access+Tokens#AccessTokens-SupportAuthenticationforNon-ExistingUsers
