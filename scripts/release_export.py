#!/usr/bin/env python3
"""Allowlisted clean export. Never inspect excluded legacy captures/history."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
ROOT_FILES = {
    'go.mod', 'go.sum', 'VERSION', 'Makefile', '.gitignore', '.dockerignore',
    '.env.example', '.gitattributes', 'README.md', 'SECURITY.md', 'CONTRIBUTING.md',
    'NOTICE.md', 'THIRD_PARTY_NOTICES.md', 'CHANGELOG.md', 'LICENSE',
}
ROOTS = {'cmd', 'internal', 'webui', 'docs', 'postman', 'deploy', 'scripts', '.github'}
DENIED = {'workspace', '.git', 'images', 'legacy', 'logs', 'data', 'secrets',
          'node_modules', '__pycache__', 'bin', 'dist', 'release-export'}
EXTENSIONS = {'.go', '.mod', '.sum', '.md', '.txt', '.html', '.css', '.js', '.json',
              '.yaml', '.yml', '.sh', '.ps1', '.py', '.svg'}
PATTERNS = {
    'private key': re.compile(r'-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----'),
    'GitHub token': re.compile(r'\b(?:gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,})\b'),
    'AWS key': re.compile(r'\bAKIA[A-Z0-9]{16}\b'),
    'Docker Hub token': re.compile(r'\bdckr_pat_[A-Za-z0-9_-]{20,}\b'),
    'private IPv4': re.compile(r'\b(?:10\.(?:\d{1,3}\.){2}\d{1,3}|192\.168\.\d{1,3}\.\d{1,3}|172\.(?:1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3})\b'),
    'embedded URL credential': re.compile(r'https?://[^\s/@:]+:[^\s/@]+@'),
}


def allowed(relative: Path) -> bool:
    parts = relative.parts
    if any(part in DENIED or part.startswith('.git-backup') for part in parts):
        return False
    if len(parts) == 1:
        return relative.name in ROOT_FILES or relative.suffix == '.go'
    if parts[0] not in ROOTS:
        return False
    if parts[0] == '.github' and (len(parts) < 3 or parts[1] != 'workflows'):
        return False
    if parts[0] == 'deploy' and (len(parts) < 3 or parts[1] != 'docker'):
        return False
    return relative.suffix in EXTENSIONS or relative.name == 'Dockerfile'


def files():
    # Walk only new-code roots; never open an excluded legacy file.
    candidates = [f for f in ROOT.iterdir() if f.is_file() and allowed(Path(f.name))]
    for name in sorted(ROOTS):
        directory = ROOT / name
        if directory.is_symlink():
            raise ValueError(f'symlink root rejected: {name}')
        if not directory.is_dir():
            continue
        stack = [directory]
        while stack:
            for item in stack.pop().iterdir():
                relative = item.relative_to(ROOT)
                if any(part in DENIED for part in relative.parts):
                    continue
                if item.is_symlink():
                    raise ValueError(f'symlink rejected: {relative.as_posix()}')
                if item.is_dir():
                    stack.append(item)
                elif allowed(relative):
                    candidates.append(item)
    return sorted(candidates, key=lambda f: f.relative_to(ROOT).as_posix())


def scan(candidates):
    errors = []
    for path in candidates:
        rel = path.relative_to(ROOT).as_posix()
        if path.is_symlink():
            errors.append(f'{rel}: symlink rejected')
            continue
        if path.stat().st_size > 4 * 1024 * 1024:
            errors.append(f'{rel}: exceeds 4 MiB text limit')
            continue
        try:
            content = path.read_text(encoding='utf-8-sig')
        except UnicodeError:
            errors.append(f'{rel}: non-UTF-8 content rejected')
            continue
        if '\0' in content:
            errors.append(f'{rel}: binary content rejected')
        for label, pattern in PATTERNS.items():
            if pattern.search(content):
                errors.append(f'{rel}: {label} detected (value withheld)')
    return errors


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path)
    parser.add_argument('--check-only', action='store_true')
    parser.add_argument('--check-tracked', action='store_true', help='Only for a clean public repository')
    args = parser.parse_args()
    if not args.check_only and args.output is None:
        parser.error('choose --check-only or --output EMPTY_DIRECTORY')
    candidates = files()
    errors = scan(candidates)
    if args.check_tracked:
        tracked = subprocess.check_output(['git', 'ls-files', '-z'], cwd=ROOT).decode().split('\0')
        for name in filter(None, tracked):
            if name != 'RELEASE_MANIFEST.json' and not allowed(Path(name)):
                errors.append(f'{name}: tracked path outside publication allowlist')
    if errors:
        print('\n'.join(errors), file=sys.stderr)
        return 1
    if args.output is not None:
        output = args.output.resolve()
        if output == ROOT or ROOT.is_relative_to(output):
            raise ValueError('output must not be the source root or its ancestor')
        if output.is_symlink() or (output.exists() and any(output.iterdir())):
            raise ValueError('output must be absent or empty; nothing is deleted automatically')
        output.mkdir(parents=True, exist_ok=True)
        manifest = {}
        for source in candidates:
            rel = source.relative_to(ROOT)
            dest = output / rel
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source, dest)
            manifest[rel.as_posix()] = hashlib.sha256(dest.read_bytes()).hexdigest()
        (output / 'RELEASE_MANIFEST.json').write_text(json.dumps(manifest, indent=2) + '\n', encoding='utf-8')
        print(f'Exported {len(manifest)} reviewed-surface files to {output}')
    else:
        print(f'Publication scan passed: {len(candidates)} allowlisted files; manual review still required')
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (ValueError, OSError, subprocess.CalledProcessError) as exc:
        print(f'Export failed: {exc}', file=sys.stderr)
        sys.exit(1)
