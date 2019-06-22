#!/usr/bin/env bash

set -e
set -o pipefail

function fail {
  echo $1 >&2
  exit 1
}

function retry {
  local n=1
  local max=15
  local delay=15
  echo -n "Waiting for $@ to succeed"
  while true; do
    "$@" >/dev/null 2>&1 && break || {
      if [[ $n -lt $max ]]; then
        ((n++))
        echo -n "."
        sleep $delay;
      else
        fail " FAILED after $n attempts!"
      fi
    }
  done
  echo " OK!"
}

function cleanup {
    kill "${vault_pid}"
}
trap cleanup EXIT

mkdir -p "${VAULT_LOG_DIR}"
vault server \
    -dev \
    -dev-plugin-dir="${VAULT_PLUGIN_DIR}" \
    -dev-plugin-init \
    -dev-root-token-id="${VAULT_TOKEN}" \
    -log-level=trace \
    > "${VAULT_LOG_DIR}/vault.log" 2>&1 &
vault_pid=$!

retry vault status

vault secrets enable \
    -path=artifactory \
    -plugin-name=artifactory \
    plugin

retry curl -fsS "${ARTIFACTORY_URL}/api/system/ping"

echo "Artifactory system/configuration:"
curl \
    -fsS \
    -u admin:password \
    -X POST \
    -H "Content-type: application/xml" \
    --data-binary @integration/artifactory.config.xml \
    "${ARTIFACTORY_URL}/api/system/configuration"
echo ""

echo "Artifactory system/security:"
curl \
    -fsS \
    -u admin:password \
    -X POST \
    -H "Content-type: application/xml" \
    --data-binary @integration/artifactory.security.xml \
    "${ARTIFACTORY_URL}/api/system/security"
echo ""

vault write artifactory/config \
    address="${ARTIFACTORY_URL}" \
    tls_verify=false \
    username=admin \
    password=password

vault write artifactory/roles/writer \
    member_of_groups=writers \
    username=vault-user-writer \
    ttl=600
WRITER_ACCESS_TOKEN="$(vault read -field=access_token artifactory/token/writer)"

vault write artifactory/roles/reader \
    member_of_groups=readers \
    username=vault-user-reader \
    ttl=600
READER_ACCESS_TOKEN="$(vault read -field=access_token artifactory/token/reader)"

echo -n "Verify writer can deploy artifact to local repository: "
if ! curl -fs -o /dev/null -X PUT -T "$0" -H "Authorization: Bearer ${WRITER_ACCESS_TOKEN}" "${ARTIFACTORY_URL}/local/artifact"
then
    echo "ERROR: writer role was unable to deploy an artifact"
    exit 1
fi
echo "SUCCESS!"

echo -n "Verify reader cannot deploy artifact to local repository: "
if curl -fs -X PUT -T "$0" -H "Authorization: Bearer ${READER_ACCESS_TOKEN}" "${ARTIFACTORY_URL}/local/artifact"
then
    echo "ERROR: reader should not be able to deploy artifact!"
    exit 1
fi
echo "SUCCESS!"

echo -n "Verify reader can read artifact: "
if ! curl -fs -o /dev/null -H "Authorization: Bearer ${READER_ACCESS_TOKEN}" "${ARTIFACTORY_URL}/local/artifact"
then
    echo "ERROR: reader role was unable to read an artifact"
    exit 1
fi
echo "SUCCESS!"

echo -n "Verify writer cannot read artifact: "
if curl -fs -o /dev/null -H "Authorization: Bearer ${WRITER_ACCESS_TOKEN}" "${ARTIFACTORY_URL}/local/artifact"
then
    echo "ERROR: writer should not be able to read artifact!"
    exit 1
fi
echo "SUCCESS!"
