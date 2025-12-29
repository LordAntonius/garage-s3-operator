#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import argparse
import os
import re
from urllib.parse import urlparse, urlunparse

try:
    import tomllib
except Exception:
    tomllib = None


def _read_toml(path: str = "/etc/garage.toml") -> dict:
    if tomllib is None:
        return {}
    try:
        with open(path, "rb") as f:
            data = tomllib.load(f)
            return data or {}
    except FileNotFoundError:
        return {}
    except Exception:
        return {}


def _assemble_url(url: str | None, port: str | None) -> str | None:
    if not url:
        return None
    u = url
    if not u.startswith(("http://", "https://")):
        u = "http://" + u
    p = urlparse(u)
    hostname = p.hostname or ""
    scheme = p.scheme or "http"
    path = p.path or ""

    if port:
        netloc = f"{hostname}:{port}"
    else:
        netloc = p.netloc or hostname

    final = urlunparse((scheme, netloc, path or "/", "", "", ""))
    return final


def parse_capacity(s: str) -> int:
    """Parse a capacity string and return bytes as integer.

    Accepts numeric value optionally followed by unit K/M/G/T/P (case-insensitive)
    and optional trailing 'B'. Uses binary multiples (1024).
    """
    if s is None:
        raise ValueError("capacity is None")
    s = str(s).strip()
    m = re.match(r"^([0-9]+(?:\.[0-9]+)?)\s*([kKmMgGtTpP]?)(?:[bB])?$", s)
    if not m:
        raise ValueError("expected number optionally followed by unit K/M/G/T/P")
    num_str, unit = m.group(1), m.group(2)
    try:
        val = float(num_str)
    except Exception:
        raise ValueError("invalid numeric value")

    if val < 0:
        raise ValueError("negative capacity")

    unit = unit.upper() if unit else ""
    mult = 1
    if unit == "K":
        mult = 1024
    elif unit == "M":
        mult = 1024 ** 2
    elif unit == "G":
        mult = 1024 ** 3
    elif unit == "T":
        mult = 1024 ** 4
    elif unit == "P":
        mult = 1024 ** 5

    bytes_val = int(val * mult)
    return bytes_val


def get_config(argv: list[str] | None = None) -> tuple[str | None, str | None, int | None]:
    """Return (final_url, token, capacity_int)

    Precedence:
      1) CLI parameters (flags --url/--port/--token/--capacity or positional url port token capacity)
      2) Environment variables (GARAGE_URL, GARAGE_PORT, GARAGE_TOKEN, GARAGE_CAPACITY)
      3) TOML file /etc/garage.toml under [admin] (url, port, admin_token)

    Note: capacity is NOT read from TOML.
    """
    parser = argparse.ArgumentParser(description="Simple API caller for garage-init")
    parser.add_argument("url_arg", nargs="?", help="API base URL (e.g. https://host or host)")
    parser.add_argument("port_arg", nargs="?", help="Port number")
    parser.add_argument("token_arg", nargs="?", help="Auth token")
    parser.add_argument("capacity_arg", nargs="?", help="Capacity (bytes) or value")
    parser.add_argument("--url", dest="url", help="API base URL")
    parser.add_argument("--port", dest="port", help="Port number")
    parser.add_argument("--token", dest="token", help="Auth token")
    parser.add_argument("--capacity", dest="capacity", help="Capacity (bytes) or value")

    args = parser.parse_args(argv)

    # CLI
    url = args.url or args.url_arg
    port = args.port or args.port_arg
    token = args.token or args.token_arg
    capacity = args.capacity or args.capacity_arg

    # env
    if not url:
        url = os.environ.get("GARAGE_URL") or os.environ.get("API_URL")
    if not port:
        port = os.environ.get("GARAGE_PORT") or os.environ.get("API_PORT")
    if not token:
        token = os.environ.get("GARAGE_TOKEN") or os.environ.get("TOKEN")
    if not capacity:
        capacity = os.environ.get("GARAGE_CAPACITY") or os.environ.get("CAPACITY")

    # toml fallback for url/port/token only
    toml_data = _read_toml("/etc/garage.toml")
    admin = toml_data.get("admin", {}) if isinstance(toml_data, dict) else {}
    if not url:
        url = admin.get("url")
    if not port:
        port_val = admin.get("port")
        if port_val is not None:
            port = str(port_val)
    if not token:
        token = admin.get("admin_token") or admin.get("token")

    final_url = _assemble_url(url, port)

    # parse capacity if present
    if capacity is not None:
        try:
            capacity_int = parse_capacity(capacity)
        except ValueError:
            raise
    else:
        capacity_int = None

    return final_url, token, capacity_int
