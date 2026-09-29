import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('review', Path(__file__).resolve().parents[1] / 'user/services/removable-media-review.py')
review = importlib.util.module_from_spec(spec)
spec.loader.exec_module(review)


class DiskReviewTests(unittest.TestCase):
    def setUp(self):
        self.disk = dict(name='sdb', path='/dev/sdb', type='disk', tran='usb',
            size=256000000000, ro=False, fstype=None, pttype=None, mountpoints=[], serial='test')

    def test_only_unformatted_unused_usb_disks_are_offered(self):
        self.assertTrue(review.eligible(self.disk))
        for changes in (dict(tran='nvme'), dict(size=0), dict(ro=True),
            dict(fstype='zfs_member'), dict(pttype='gpt'), dict(children=[{}]),
            dict(mountpoints=['/media/data']), dict(type='part')):
            with self.subTest(changes=changes):
                self.assertFalse(review.eligible(self.disk | changes))

    def test_replaced_device_is_rejected_even_with_same_path_and_serial(self):
        with patch.object(review, 'disks', return_value=[self.disk]), \
             patch.object(review, 'identity', return_value=('new disk sequence',)):
            with self.assertRaisesRegex(RuntimeError, 'changed'):
                review.revalidate(self.disk, ('old disk sequence',), blank=True)

    def test_new_filesystem_after_dialog_cancels_format(self):
        with patch.object(review, 'disks', return_value=[self.disk | dict(fstype='ext4')]), \
             patch.object(review, 'identity', return_value=('same',)):
            with self.assertRaisesRegex(RuntimeError, 'no longer'):
                review.revalidate(self.disk, ('same',), blank=True)

    def test_no_format_call_if_final_identity_check_fails(self):
        with patch.object(review, 'call', return_value=[[True, '']]) as call, \
             patch.object(review, 'revalidate', side_effect=RuntimeError('changed')):
            with self.assertRaises(RuntimeError):
                review.setup(self.disk, ('old',))
            self.assertEqual([c.args[2] for c in call.call_args_list], ['CanFormat'])

    def test_unknown_layout_after_gpt_stops_before_partition_format(self):
        with patch.object(review, 'call', return_value=[[True, '']]) as call, \
             patch.object(review, 'revalidate', side_effect=[self.disk, self.disk | dict(pttype='gpt', children=[{}])]), \
             patch.object(review, 'run'):
            with self.assertRaisesRegex(RuntimeError, 'Unexpected'):
                review.setup(self.disk, ('same',))
            self.assertEqual([c.args[2] for c in call.call_args_list], ['CanFormat', 'Format'])


if __name__ == '__main__':
    unittest.main()
