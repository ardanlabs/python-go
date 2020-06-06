import ctypes
from pathlib import Path
from distutils.sysconfig import get_config_var

ext_suffix = get_config_var('EXT_SUFFIX')

here = Path(__file__).absolute().parent
so_file = here / ('_checksig' + ext_suffix)
so = ctypes.cdll.LoadLibrary(so_file)

verify = so.verify
verify.argtypes = [ctypes.c_char_p]
verify.restype = ctypes.c_char_p


class SignatureError(Exception):
    pass


def check_signature(root_dir):
    res = verify(root_dir.encode('utf-8'))
    if res is not None:
        msg = res.decode('utf-8')
        raise SignatureError(msg)
