# Artifactory Secrets Engine (API)

This is the API documentation for the Vault Artifactory secrets engine.

This documentation assumes the Artifactory secrets engine is enabled at the `/artifactory` path in Vault. Since it is possible to enable secrets engines at any location, please update your API calls accordingly.

## Configure Access

This endpoint configures the access information for Artifactory. This access information is used so that Vault can communicate with Artifactory and generate Artifactory access tokens.

| Method | Path |
|:-------|:-----|
|`POST`  | `/artifactory/config` |

### Paramaters

 * `address` `(string: required)` - Specifies the Artifactory URL, e.g. `https://artifactory.example.com/artifactory`
 * `api_key` `(string: required)` - The API key associated with the (optinally admin) user which will be used to generate access tokens.


### Sample Payload

```json
{
    "address": "https://artifactory.example.com/artifactory",
    "api_key": "AKCp5ZkK11XnHiqJ1mFgivc1NePCXXE2Ujk9jGHhPp4K4XqMp25bpoSFeFwn6ExSBXy7n7uw9"
}
```

## Create/Update Role

This endpoint creates/updates an Artifactory role definition.  If the role does not exist, it will be created. If the role already exists, it will receive updated attributes.

| Method | Path |
|:-------|:-----|
|`POST`  | `/artifactory/roles/:name` |

### Paramaters

 * `name` `(string: required)` - Specifies the name of an existing role against which to create this Artifactory access token. This is part of the request URL.
 * `username` `(string: optional)` - The user name for which this token is created. If the user does not exist, a transient user is created. Non-admin users can only create tokens for themselves so they must specify their own username. If the user does not exist, the `member_of_groups` must be provided.
 * `member_of_groups` `(list: <group name>)` - The list of groups that the token is associated with. Translates to `scope=member-of-groups:...`.
 * `ttl` `(duration="")` - Specifies the TTL for this role. This is provided as a string duration with a time suffix like "30s" or "1h" or as seconds. If not provided, the default Vault TTL is used.


### Sample Payload

```json
{
    "username": "rt-user",
    "member_of_groups": [
        "Readers",
        "Group with spaces"
    ],
    "ttl": "1h"
}
```

## Read Role

This endpoint queries for information about a Artifactory role with the given name. If no role exists with that name, a 404 is returned.

| Method | Path |
|:-------|:-----|
|`GET`  | `/artifactory/roles/:name` |

### Paramaters

 * `name` `(string: required)` - Specifies the name of the role to query. This is part of the request URL.

## List Roles

This endpoint lists all existing roles in the secrets engine.

| Method | Path |
|:-------|:-----|
|`LIST`  | `/artifactory/roles` |

## Delete Role

This endpoint lists all existing roles in the secrets engine.

| Method | Path |
|:-------|:-----|
|`DELETE`  | `/artifactory/roles/:name` |

### Paramaters

 * `name` `(string: required)` - Specifies the name of the role to delete. This is part of the request URL. 

## Create Access Token

This endpoint creates an Artifactory access token based on the given role definition.


| Method | Path |
|:-------|:-----|
|`GET`   | `/artifactory/token/:name` |

### Paramaters

 * `name` `(string: required)` - Specifies the name of an existing role against which to create this Artifactory access token. This is part of the request URL. 

### Sample Response

```json
{
    "data": {
        "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
}
```
