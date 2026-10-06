import struct
from pathlib import Path
import tempfile
import unittest
from scripts.release.release_lib.pe_imports import imports


class PeImportTests(unittest.TestCase):
    def fixture(self, directory):
        data = bytearray(2048)
        data[:2] = b'MZ'
        struct.pack_into('<I', data, 0x3c, 0x80)
        data[0x80:0x84] = b'PE\0\0'
        struct.pack_into('<HH', data, 0x84, 0x8664, 1)
        struct.pack_into('<H', data, 0x94, 240)
        optional = 0x98
        struct.pack_into('<H', data, optional, 0x20b)
        struct.pack_into('<II', data, optional + 120, 0x1000, 40)
        struct.pack_into('<II', data, optional + 216, 0x1080, 64)
        struct.pack_into('<IIII', data, optional + 240 + 8, 1536, 0x1000, 1536, 512)
        struct.pack_into('<I', data, 512 + 12, 0x1100)
        struct.pack_into('<II', data, 640, 1, 0x1120)
        data[768:781] = b'KERNEL32.dll\0'
        data[800:814] = b'libcodec.dll\0\0'
        file = directory / 'fixture.exe'
        file.write_bytes(data)
        return file

    def test_reads_normal_and_delayed_dependencies(self):
        with tempfile.TemporaryDirectory() as temporary:
            file = self.fixture(Path(temporary))
            self.assertEqual(imports(file), {'kernel32.dll', 'libcodec.dll'})

    def test_rejects_wrong_architecture_and_unknown_delayed_format(self):
        with tempfile.TemporaryDirectory() as temporary:
            file = self.fixture(Path(temporary))
            data = bytearray(file.read_bytes())
            struct.pack_into('<H', data, 0x84, 0x14c)
            file.write_bytes(data)
            with self.assertRaisesRegex(ValueError, 'x64'):
                imports(file)
            file = self.fixture(Path(temporary))
            data = bytearray(file.read_bytes())
            struct.pack_into('<I', data, 640, 0)
            file.write_bytes(data)
            with self.assertRaisesRegex(ValueError, 'VA-form'):
                imports(file)
