"""Read x64 PE imports without executing third-party code or relying on PATH."""
from __future__ import annotations
import struct
from pathlib import Path


def imports(file: Path) -> set[str]:
    data = file.read_bytes()
    if data[:2] != b'MZ':
        raise ValueError(f'Not a PE file: {file.name}')
    pe = struct.unpack_from('<I', data, 0x3c)[0]
    if data[pe:pe + 4] != b'PE\0\0':
        raise ValueError('Invalid PE signature')
    machine, count = struct.unpack_from('<HH', data, pe + 4)
    optional_size = struct.unpack_from('<H', data, pe + 20)[0]
    optional = pe + 24
    if machine != 0x8664 or struct.unpack_from('<H', data, optional)[0] != 0x20b:
        raise ValueError('Native engine requires x64 PE32+')
    sections = optional + optional_size

    def offset(rva):
        for index in range(count):
            virtual_size, virtual, size, raw = struct.unpack_from('<IIII', data, sections + index * 40 + 8)
            if virtual <= rva < virtual + max(size, virtual_size):
                return raw + rva - virtual
        raise ValueError(f'PE RVA outside sections: {rva}')

    result = set()
    # IMAGE_IMPORT_DESCRIPTOR and IMAGE_DELAYLOAD_DESCRIPTOR (RVA form).
    for directory, stride, name_offset in ((1, 20, 12), (13, 32, 4)):
        rva, size = struct.unpack_from('<II', data, optional + 112 + directory * 8)
        if not rva or not size:
            continue
        position = offset(rva)
        for _ in range(size // stride):
            block = data[position:position + stride]
            if not any(block):
                break
            if directory == 13 and not struct.unpack_from('<I', block)[0] & 1:
                raise ValueError('Unsupported VA-form delayed imports')
            name = offset(struct.unpack_from('<I', block, name_offset)[0])
            end = data.index(0, name)
            result.add(data[name:end].decode('ascii').lower())
            position += stride
    return result
