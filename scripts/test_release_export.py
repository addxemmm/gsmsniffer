"""Synthetic unit tests for publication boundaries; no private legacy data read."""
import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('release_export', Path(__file__).with_name('release_export.py'))
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class ExportTests(unittest.TestCase):
    def test_allowed_paths(self):
        for name in ['README.md', 'cmd/server/main.go', 'internal/webui/index.html',
                     'docs/API.md', 'deploy/docker/Dockerfile', '.github/workflows/ci.yml']:
            self.assertTrue(release.allowed(Path(name)), name)
        for name in ['workspace/source.py', 'images/report.svg', '.git/config', '.env',
                     'docs/legacy/readme.md', 'internal/sample.pcap', 'secrets/api_token',
                     'deploy/private.yml', 'scripts/__pycache__/cache.py']:
            self.assertFalse(release.allowed(Path(name)), name)

    def test_scan_rejects_credentials_without_printing_them(self):
        with tempfile.TemporaryDirectory() as temp:
            previous = release.ROOT
            try:
                release.ROOT = Path(temp)
                file = release.ROOT / 'README.md'
                secret = 'ghp_' + 'A' * 36
                file.write_text(secret, encoding='utf-8')
                errors = release.scan([file])
                self.assertTrue(errors)
                self.assertNotIn(secret, '\n'.join(errors))
                file.write_text('TOKEN is a placeholder. http://127.0.0.1:8080', encoding='utf-8')
                self.assertEqual(release.scan([file]), [])
            finally:
                release.ROOT = previous

    def test_legacy_files_are_not_selected(self):
        with tempfile.TemporaryDirectory() as temp:
            previous = release.ROOT
            try:
                release.ROOT = Path(temp)
                (release.ROOT / 'README.md').write_text('new source', encoding='utf-8')
                (release.ROOT / 'workspace').mkdir()
                (release.ROOT / 'workspace' / 'private.log').write_bytes(b'\xff\x00')
                self.assertEqual([p.name for p in release.files()], ['README.md'])
            finally:
                release.ROOT = previous


if __name__ == '__main__':
    unittest.main()
