import contextlib
import io
import unittest
from unittest.mock import patch
import hub_tags

DIGEST = 'sha256:' + 'a' * 64


class ConsolidationTests(unittest.TestCase):
    def run_script(self, tags, apply=False, expected=DIGEST):
        calls = []
        def fake(url, method='GET', token=None, body=None):
            calls.append((url, method))
            if url.endswith('/auth/token'):
                return {'access_token': 'test-only'}
            if method == 'DELETE':
                del tags[url.rsplit('/', 1)[1]]
                return
            if url.endswith('/2.1'):
                return tags['2.1']
            return {'results': list(tags.values()), 'next': None}
        argv = ['hub_tags.py', '--expected-digest', expected] + (['--apply'] if apply else [])
        with patch('sys.argv', argv), patch.dict('os.environ', {'HUB_USER': 'addxemmm', 'HUB_TOKEN': 'test-only'}), patch.object(hub_tags, 'request', side_effect=fake), contextlib.redirect_stdout(io.StringIO()):
            hub_tags.main()
        return calls

    def test_default_does_not_delete(self):
        calls = self.run_script({'2.1': {'name': '2.1', 'digest': DIGEST}})
        self.assertTrue(all(method == 'GET' for _, method in calls))

    def test_only_known_legacy_tags_deleted(self):
        tags = {name: {'name': name, 'digest': DIGEST} for name in hub_tags.OLD | {'2.1'}}
        calls = self.run_script(tags, apply=True)
        self.assertEqual(set(tags), {'2.1'})
        self.assertEqual(sum(method == 'DELETE' for _, method in calls), 4)

    def test_missing_or_changed_retained_tag_fails(self):
        for tags in [{}, {'2.1': {'name': '2.1', 'digest': 'different'}}]:
            with self.assertRaises(ValueError):
                self.run_script(tags, apply=True)

    def test_unexpected_tag_fails(self):
        with self.assertRaises(ValueError):
            self.run_script({'2.1': {'name': '2.1', 'digest': DIGEST}, 'other': {'name': 'other'}}, apply=True)


if __name__ == '__main__':
    unittest.main()
