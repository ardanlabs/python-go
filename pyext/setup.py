from distutils.errors import CompileError
from subprocess import call

from setuptools import Extension, setup
from setuptools.command.build_ext import build_ext


class build_go_ext(build_ext):
    def build_extension(self, ext):
        ext_path = self.get_ext_fullpath(ext.name)
        cmd = ['go', 'build', '-buildmode=c-shared', '-o', ext_path]
        cmd += ext.sources
        out = call(cmd)
        if out != 0:
            raise CompileError('Go build failed')


setup(
    name='checksig',
    version='0.1.0',
    py_modules=['checksig'],
    ext_modules=[
        Extension('_checksig', ['checksig.go', 'export.go'])
    ],
    cmdclass={'build_ext': build_go_ext},
)
