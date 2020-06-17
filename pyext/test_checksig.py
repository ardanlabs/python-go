from checksig import check_signatures

from unittest import TestCase


class TestCheckSignatures(TestCase):
    def test_logs(self):
        logs_dir = 'testdata/logs'
        with self.assertRaises(ValueError):
            check_signatures(logs_dir)
