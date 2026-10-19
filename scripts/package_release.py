#!/usr/bin/env python3
"""Package a committed tree and verify its public binary supplement.

Source archives contain exactly git-tracked files at the chosen immutable ref.
The external source manifest hashes each file, avoiding self-hash cycles.
This command never changes tags, frozen artifacts, or an existing supplement.
"""
import argparse
import gzip
import hashlib
import io
import json
import pathlib
import subprocess
import tarfile

ROOT = pathlib.Path(__file__).resolve().parents[1]

def git(*args):
    return subprocess.check_output(['git', '-C', str(ROOT), *args])

def sha(data):
    return hashlib.sha256(data).hexdigest()

def verify_supplement(path, ref):
    # Use the tagged manifest rather than possibly edited working-tree metadata.
    manifest = json.loads(git('show', f'{ref}:docs/EVIDENCE_MANIFEST.json'))
    expected = {entry['path']: entry for entry in manifest['files'] if entry['supplement']}
    seen = set()
    with tarfile.open(path, 'r:*') as archive:
        for member in archive:
            if member.isdir():
                continue
            if not member.isfile() or member.name not in expected:
                raise ValueError(f'Unexpected supplement member: {member.name}')
            if member.name in seen:
                raise ValueError(f'Duplicate supplement member: {member.name}')
            entry = expected[member.name]
            assert member.size == entry['bytes'], member.name
            stream = archive.extractfile(member)
            digest = hashlib.sha256()
            for block in iter(lambda: stream.read(1 << 20), b''):
                digest.update(block)
            assert digest.hexdigest() == entry['sha256'], member.name
            seen.add(member.name)
    assert seen == set(expected), f'Missing members: {set(expected)-seen}'
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1 << 20), b''):
            digest.update(block)
    return dict(path=path.name, bytes=path.stat().st_size, sha256=digest.hexdigest(), checked_binary_files=len(seen))

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--ref', required=True, help='Immutable release tag or exact commit')
    parser.add_argument('--out', type=pathlib.Path, default=ROOT/'dist')
    parser.add_argument('--source', action='store_true', help='Write deterministic exact Git source tar.gz and external file manifest')
    parser.add_argument('--verify-supplement', type=pathlib.Path)
    args = parser.parse_args()
    commit = git('rev-parse', f'{args.ref}^{{commit}}').decode().strip()
    report = dict(ref=args.ref, commit=commit)
    if args.source:
        prefix = 'fhe-quadratic-ckks-' + args.ref.replace('/', '-') + '/'
        raw = git('archive', '--format=tar', f'--prefix={prefix}', commit)
        entries = []
        with tarfile.open(fileobj=io.BytesIO(raw), mode='r:') as archive:
            for member in archive:
                if member.isfile():
                    data = archive.extractfile(member).read()
                    entries.append(dict(path=member.name.removeprefix(prefix), bytes=len(data), sha256=sha(data)))
                elif not member.isdir():
                    raise ValueError(f'Unsupported source member: {member.name}')
        args.out.mkdir(parents=True, exist_ok=True)
        archive_path = args.out/(prefix.rstrip('/')+'.tar.gz')
        if archive_path.exists():
            raise FileExistsError(f'Refusing to replace existing archive: {archive_path}')
        with archive_path.open('xb') as output:
            with gzip.GzipFile(filename='', fileobj=output, mode='wb', mtime=0) as compressed:
                compressed.write(raw)
        payload = dict(commit=commit, ref=args.ref, archive=archive_path.name, prefix=prefix,
                       sha256=sha(archive_path.read_bytes()), bytes=archive_path.stat().st_size,
                       files=entries)
        manifest_path = args.out/(prefix.rstrip('/')+'-source-manifest.json')
        manifest_path.write_text(json.dumps(payload, indent=2)+'\n')
        report['source'] = {k: v for k, v in payload.items() if k != 'files'}
        report['source']['file_count'] = len(entries)
        report['source']['manifest'] = manifest_path.name
    if args.verify_supplement:
        report['supplement'] = verify_supplement(args.verify_supplement, commit)
    if not args.source and not args.verify_supplement:
        parser.error('Choose --source and/or --verify-supplement')
    print(json.dumps(report, indent=2))

if __name__ == '__main__':
    main()
