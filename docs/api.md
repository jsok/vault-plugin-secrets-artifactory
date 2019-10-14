---
layout: docs
---

{::options toc_levels="2" /}

# Artifactory Database Plugin HTTP API

The Artifactory database plugin is one of the supported plugins for the database
secrets engine. This plugin generates credentials dynamically based on
configured roles for Artifactory.

## Configure Connection

In addition to the parameters defined by the [Database
Backend](/api/secret/databases/index.html#configure-connection), this plugin
has a number of parameters to further configure a connection.

| Method   | Path                         |
| :--------------------------- | :--------------------- |
| `POST`   | `/database/config/:name`     |

### Parameters

- `addres` `(string: <required>)` - The URL for Artifactory's API ("http://localhost:8081/artifactory").
- `username` `(string: "")` - The username to be used for Artifactory ("vault"). Required if using basic auth.
- `password` `(string: "")` - The password to be used for Artifactory ("pa55w0rd"). Required if using basic auth.
- `api_key` `(string: "")` - The access token to be used for Artifactory. Either username and password or the api key are required to authenticate to Artifactory.
- `insecure` `(bool: false)` - Not recommended. Default to false. Can be set to true to disable SSL verification.

### Sample Payload

```json
{
  "plugin_name": "artifactory-database-plugin",
  "allowed_roles": "internally-defined-role,externally-defined-role",
  "url": "http://localhost:9200",
  "username": "vault",
  "password": "myPa55word",
}
```

### Sample Request

```
$ curl \
    --header "X-Vault-Token: ..." \
    --request POST \
    --data @payload.json \
    http://127.0.0.1:8200/v1/database/config/my-artifactory-database
```

## Statements

Statements are configured during role creation and are used by the plugin to
determine what is sent to the database on user creation, renewing, and
revocation. For more information on configuring roles see the [Role
API](/api/secret/databases/index.html#create-role) in the database secrets engine docs.

### Parameters

The following are the statements used by this plugin. If not mentioned in this
list the plugin does not support that statement type.

- `creation_statements` `(string: <required>)` – Using JSON, list
  `artifactory_groups` whose Artifactory permissions the role should adopt.
  They must pre-exist in Artifactory.

### Sample Creation Statements
```json
{
  "artifactory_groups": ["pre-existing-group-in-artifactory"]
}
```
