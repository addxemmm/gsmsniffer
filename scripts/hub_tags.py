#!/usr/bin/env python3
"""One-time, digest-guarded migration to the single 2.1 Docker Hub tag."""
import argparse
import json
import os
import re
import urllib.error
import urllib.request

BASE = 'https://hub.docker.com/v2/namespaces/addxemmm/repositories/gsmsniffer/tags'
OLD = {'2.0.0', '2.0.0-shielded', '2.0.0-sha-836187ff0940',
       '2.0.0-sha-836187ff0940-shielded'}


def request(url, method='GET', token=None, body=None):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = 'Bearer ' + token
    data = json.dumps(body).encode() if body is not None else None
    with urllib.request.urlopen(urllib.request.Request(
            url, data=data, headers=headers, method=method), timeout=45) as response:
        raw = response.read()
        return json.loads(raw) if raw else None


def inventory(token=None):
    page = request(BASE + '?page_size=100', token=token)
    # This migration expects five or fewer tags. Fail closed on pagination.
    if page.get('next'):
        raise ValueError('Unexpected paginated inventory; manual review required')
    tags = {item['name']: item for item in page['results']}
    if set(tags) - OLD - {'2.1'}:
        raise ValueError('Unexpected tags; no changes made')
    return tags


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--expected-digest', required=True)
    parser.add_argument('--apply', action='store_true')
    args = parser.parse_args()
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', args.expected_digest):
        raise ValueError('Expected digest must be SHA-256')
    tags = inventory()
    if tags.get('2.1', {}).get('digest') != args.expected_digest:
        raise ValueError('2.1 digest missing or changed; no changes made')
    print('Keep 2.1:', args.expected_digest)
    print('Old tags:', ', '.join(sorted(set(tags) & OLD)))
    if not args.apply:
        return
    if os.environ['HUB_USER'] != 'addxemmm':
        raise ValueError('Account mismatch')
    auth = request('https://hub.docker.com/v2/auth/token', 'POST', body={
        'identifier': os.environ['HUB_USER'], 'secret': os.environ['HUB_TOKEN']})
    token = auth['access_token']
    for tag in sorted(set(tags) & OLD):
        if request(BASE + '/2.1', token=token)['digest'] != args.expected_digest:
            raise ValueError('2.1 changed; cleanup stopped')
        request(BASE + '/' + tag, 'DELETE', token=token)
        print('Deleted tag:', tag)
    remaining = inventory(token)
    if set(remaining) != {'2.1'} or remaining['2.1']['digest'] != args.expected_digest:
        raise ValueError('Final registry inventory did not match; inspect manually')
    print('Verified: only 2.1 remains')


if __name__ == '__main__':
    try:
        main()
    except urllib.error.HTTPError as exc:
        raise SystemExit(f'Docker Hub HTTP {exc.code}; no credentials logged') from None
