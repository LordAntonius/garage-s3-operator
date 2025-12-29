#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import sys
import requests
from config import get_config


def get_cluster_status(base_url: str, token: str | None = None) -> dict:
    """Call the admin API GET /v2/GetClusterStatus and return the parsed JSON object.

    Parameters:
      - base_url: base URL including scheme and optional port (e.g. http://host:3903/)
      - token: optional bearer token to send as Authorization header

    Raises requests.HTTPError on non-2xx responses or requests.RequestException on network errors.
    """
    if not base_url:
        raise ValueError("base_url is required")
    # Ensure no trailing slash to avoid double slashes
    endpoint = base_url.rstrip("/") + "/v2/GetClusterStatus"
    headers = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    resp = requests.get(endpoint, headers=headers, timeout=10)
    resp.raise_for_status()
    return resp.json()


def update_cluster_layout(base_url: str, token: str | None, roles: list) -> dict:
    """Call the admin API POST /v2/UpdateClusterLayout with given roles.

    Parameters:
      - base_url: base URL including scheme and optional port (e.g. http://host:3903/)
      - token: optional bearer token
      - roles: list of role objects to send under the `roles` field

    Returns the parsed JSON response. Raises requests.HTTPError on non-2xx.
    """
    if not base_url:
        raise ValueError("base_url is required")
    endpoint = base_url.rstrip("/") + "/v2/UpdateClusterLayout"
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    payload = {
        "roles": roles
    }

    resp = requests.post(endpoint, json=payload, headers=headers, timeout=30)
    resp.raise_for_status()
    return resp.json()


def apply_cluster_layout(base_url: str, token: str | None) -> dict:
    """Call the admin API POST /v2/ApplyClusterLayout with version=1.

    Parameters:
      - base_url: base URL including scheme and optional port (e.g. http://host:3903/)
      - token: optional bearer token

    Returns the parsed JSON response. Raises requests.HTTPError on non-2xx.
    """
    if not base_url:
        raise ValueError("base_url is required")
    endpoint = base_url.rstrip("/") + "/v2/ApplyClusterLayout"
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    payload = {"version": 1}

    resp = requests.post(endpoint, json=payload, headers=headers, timeout=30)
    resp.raise_for_status()
    return resp.json()

def main():
    # Delegate param retrieval to get_config()
    url, token, capacity = get_config()
    if not url:
        print("No URL/port provided. Provide via CLI, env vars, or /etc/garage.toml", file=sys.stderr)
        sys.exit(2)
    if not capacity:
        print("No capacity provided. Provide via CLI or env vars", file=sys.stderr)
        sys.exit(2)

    try:
        # Wait loop for nodes status to be all in isUp == true
        isClusterReady = False
        while not isClusterReady:
            status = get_cluster_status(url, token)
            nodes = status.get("nodes", [])
            isClusterReady = all(node.get("isUp", False) for node in nodes)
            if not isClusterReady:
                print("Waiting for all nodes to be up...")
                import time
                time.sleep(5)

        # if layoutVersion != 0, cluster is already initialized, nothing to do
        if status["layoutVersion"] != 0:
            print("Cluster is already initialized.")
            sys.exit(0)

        # Roles will contain an array of dictionary with fields capacity, tags, zone and id
        roles = []
        for node in status["nodes"]:
            # Create role for each node
            node_role = {
                "capacity": capacity,           # given by get_config
                "tags": [node.get("hostname")], # singleton list with hostname
                "zone": "garage",               # fixed zone name
                "id": node.get("id"),           # node id
            }
            roles.append(node_role)
        
        # Apply the new cluster layout
        resp = update_cluster_layout(url, token, roles)

        # Apply the layout changes
        resp = apply_cluster_layout(url, token)
        print("Applied layout on", len(roles), "nodes.")
        for m in resp["message"]:
            print(m)
    except requests.HTTPError as e:
        print(f"HTTP error while calling API: {e}", file=sys.stderr)
        sys.exit(3)

if __name__ == "__main__":
    main()
