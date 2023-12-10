#!/usr/bin/env python

from itertools import product
from subprocess import call, STDOUT

version = '0.5'
progname = 'bodash'
go_os = ['darwin', 'linux', 'windows']
go_arch = ['amd64', 'arm64']

targets = list(product(go_os, go_arch))
for target_os, target_arch in targets:
    outname = f'{progname}-{version}-{target_os}_{target_arch}'
    outdir = 'dist'
    outdir_full = f'{outdir}/{outname}'

    call(['mkdir', '-vp', outdir_full], stderr=STDOUT)
    call(f'GOOS={target_os} GOARCH={target_arch} go build -o {outdir_full}', shell=True, stderr=STDOUT)
    if target_os == 'windows':
        call(['zip', '-rv', f'{outname}.zip', outname], cwd=outdir, stderr=STDOUT)
    else:
        call(['tar', 'czvf', f'{outname}.tar.gz', outname], cwd=outdir, stderr=STDOUT)
