import argparse
import ipaddress
import json
import os
import shutil
import subprocess
from pathlib import Path

import certifi
import psycopg


PROJECT = Path(__file__).resolve().parents[1]
ROLE = "petclinic_app"


def command(tool, *args):
    executable = shutil.which(tool)
    if executable is None:
        raise RuntimeError(f"Required tool is not installed: {tool}")
    result = subprocess.run(
        [executable, *args], cwd=PROJECT, shell=os.name == "nt",
        capture_output=True, text=True, timeout=120,
    )
    if result.returncode:
        raise RuntimeError(f"{tool} failed: {result.stderr}")
    return result.stdout


def connect(env, database):
    token = json.loads(command(
        "az", "account", "get-access-token", "--resource-type", "oss-rdbms", "--output", "json",
    ))["accessToken"]
    credentials = {"password": token}
    return psycopg.connect(
        host=env["POSTGRES_HOST"], port=5432, dbname=database,
        user=env["POSTGRES_ADMIN_NAME"], **credentials,
        sslmode="verify-full", sslrootcert=certifi.where(),
        connect_timeout=20, autocommit=True,
    )


def bootstrap(env):
    if env.get("PROVISION_APP", "false").lower() != "true":
        with connect(env, "petclinic") as connection:
            connection.execute("SELECT 1").fetchone()
        print("Database-only preflight: Entra administrator connection succeeded; app bootstrap deferred.")
        return

    app = json.loads(command("az", "containerapp", "show", "--ids", env["AZURE_RESOURCE_PETCLINIC_ID"], "-o", "json"))
    principal = app["identity"]["principalId"]
    if app["identity"]["type"] != "SystemAssigned" or principal != env["PETCLINIC_PRINCIPAL_ID"]:
        raise RuntimeError("Expected exactly the provisioned system-assigned identity")
    outbound = app["properties"].get("outboundIpAddresses", [])
    if not outbound:
        raise RuntimeError("No app outbound addresses; refusing to open an unrestricted database firewall")
    for address in sorted(set(outbound)):
        ip = str(ipaddress.IPv4Address(address))
        command(
            "az", "postgres", "flexible-server", "firewall-rule", "create",
            "--resource-group", env["AZURE_RESOURCE_GROUP"],
            "--server-name", env["POSTGRES_SERVER_NAME"], "--name", "aca-" + ip.replace(".", "-"),
            "--start-ip-address", ip, "--end-ip-address", ip, "--only-show-errors",
        )

    with connect(env, "postgres") as connection:
        existing = connection.execute(
            "SELECT objectid FROM pg_catalog.pgaadauth_list_principals(false) WHERE rolname = %s", (ROLE,),
        ).fetchone()
        if existing:
            if str(existing[0]).lower() != principal.lower():
                raise RuntimeError("Existing database role is bound to a different identity")
        else:
            connection.execute(
                "SELECT pg_catalog.pgaadauth_create_principal_with_oid(%s, %s, 'service', false, false)",
                (ROLE, principal),
            )

    with connect(env, "petclinic") as connection:
        connection.execute("REVOKE ALL ON DATABASE petclinic FROM PUBLIC")
        connection.execute("REVOKE CREATE ON SCHEMA public FROM PUBLIC")
        schema = PROJECT / "src" / "main" / "resources" / "db" / "postgres"
        connection.execute((schema / "schema.sql").read_text(), prepare=False)
        connection.execute((schema / "data.sql").read_text(), prepare=False)
        connection.execute("GRANT CONNECT ON DATABASE petclinic TO petclinic_app")
        connection.execute("GRANT USAGE ON SCHEMA public TO petclinic_app")
        connection.execute(
            "GRANT SELECT ON vets, specialties, vet_specialties, types, owners, pets, visits TO petclinic_app"
        )
        connection.execute("GRANT INSERT, UPDATE ON owners, pets, visits TO petclinic_app")
        connection.execute("GRANT USAGE ON SEQUENCE owners_id_seq, pets_id_seq, visits_id_seq TO petclinic_app")
    command(
        "az", "containerapp", "registry", "set",
        "--ids", env["AZURE_RESOURCE_PETCLINIC_ID"],
        "--server", env["AZURE_CONTAINER_REGISTRY_ENDPOINT"], "--identity", "system",
        "--only-show-errors",
    )
    command(
        "az", "containerapp", "ingress", "update", "--ids", env["AZURE_RESOURCE_PETCLINIC_ID"],
        "--target-port", "8080", "--only-show-errors",
    )
    command("azd", "env", "set", "APP_RUNTIME_READY", "true")
    print("System-assigned identity mapped to non-admin SQL role; schema bootstrapped; scoped grants applied.")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=["bootstrap"])
    parser.parse_args()
    bootstrap(json.loads(command("azd", "env", "get-values", "--output", "json")))
