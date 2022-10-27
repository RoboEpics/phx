from os import path, getcwd, environ
import tarfile
cwd = getcwd()
with tarfile.open(environ['TAR_FILE'], 'x:gz') as t:
    t.add(cwd, path.basename(cwd))
