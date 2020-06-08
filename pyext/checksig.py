"""Parallel check of files digital signature"""

import ctypes
from pathlib import Path
from distutils.sysconfig import get_config_var

# Find out where the shared library is and
ext_suffix = get_config_var('EXT_SUFFIX')
here = Path(__file__).absolute().parent
so_file = here / ('_checksig' + ext_suffix)


# Load function and set its signature
so = ctypes.cdll.LoadLibrary(so_file)
verify = so.verify
verify.argtypes = [ctypes.c_char_p]
verify.restype = ctypes.c_void_p
free = so.free
free.argtypes = [ctypes.c_void_p]


class SignatureError(Exception):
    """SignatureError is an error in signature"""
    pass


def check_signature(root_dir):
    """Check (in parallel) digital signature of all files in root_dir.
    We assume there's a sha1sum.txt file under root_dir
    """
    res = verify(root_dir.encode('utf-8'))
    if res is not None:
        msg = ctypes.string_at(res).decode('utf-8')
        free(res)
        raise SignatureError(msg)
