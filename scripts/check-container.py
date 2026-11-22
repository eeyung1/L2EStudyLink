#!/usr/bin/env python3
"""Verify that the built image boots and serves runtime templates/assets in CI."""
import json
import sys
import time
import urllib.error
import urllib.request

origin = sys.argv[1].rstrip('/')
last_error = None
for attempt in range(45):
    try:
        with urllib.request.urlopen(origin + '/health', timeout=2) as response:
            payload = json.load(response)
        if payload.get('status') != 'ok':
            raise ValueError('Unexpected health response')
        break
    except (OSError, ValueError, urllib.error.URLError) as error:
        last_error = error
        time.sleep(1)
else:
    raise SystemExit(f'Compiled container did not start: {last_error}')
for path, expected in [('/dashboard', b'L2EStudyLink'), ('/static/css/workspace.css', b'l2e-workspace')]:
    with urllib.request.urlopen(origin + path, timeout=5) as response:
        if expected not in response.read():
            raise SystemExit(f'Runtime asset missing: {path}')
print('Compiled container started against PostgreSQL and served health, templates and assets.')
