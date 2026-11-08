#!/usr/bin/env python3
"""Package static Linux release binaries and their example configuration."""
import argparse
import hashlib
from pathlib import Path
import tarfile

parser = argparse.ArgumentParser()
parser.add_argument('--version', required=True)
args = parser.parse_args()
root = Path(__file__).resolve().parents[2]
out = root / 'dist' / 'releases' / args.version
out.mkdir(parents=True, exist_ok=True)
checksums = []
for service in ('frontier', 'frontlas'):
    for arch in ('amd64', 'arm64'):
        target = out / f'{service}-{args.version}-linux-{arch}.tar.gz'
        with tarfile.open(target, 'w:gz') as archive:
            archive.add(root / 'dist/bin' / f'{service}-linux-{arch}', arcname=service)
            archive.add(root / 'etc' / f'{service}.yaml', arcname=f'{service}.yaml')
        checksums.append(f'{hashlib.sha256(target.read_bytes()).hexdigest()}  {target.name}\n')
(out / 'SHA256SUMS').write_text(''.join(checksums))
print(out)
