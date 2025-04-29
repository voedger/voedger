#!/usr/bin/env python3
import pathlib, re, sys

BASE   = 322680000131072
START  = 200001
PAT    = re.compile(r'322680000(\d{6})')
SUFFIX = {'.go'}

def replacement(m: re.Match) -> str:
    return str(START + int(m.group(0)) - BASE)

def read_with_encoding(path):
    """Try a few encodings; return (text, encoding) or (None, None)."""
    for enc in ('utf-8', 'latin-1', 'cp1252'):
        try:
            with path.open('r', encoding=enc, newline='') as f:   # ← keep original EOLs
                return f.read(), enc
        except UnicodeDecodeError:
            continue
    print(f'skipping {path}: unknown encoding', file=sys.stderr)
    return None, None

for p in pathlib.Path('.').rglob('*'):
    if p.suffix.lower() not in SUFFIX:
        continue

    text, enc = read_with_encoding(p)
    if text is None:
        continue

    new = PAT.sub(replacement, text)
    if new != text:
        with p.open('w', encoding=enc, newline='') as f:          # ← write them back unchanged
            f.write(new)
        print(f'updated {p}')