"""Неподписанный артефакт CI. Ключи владельца в GitHub не передаются."""
from pathlib import Path
import sys, zipfile
source,target=map(Path,sys.argv[1:])
with zipfile.ZipFile(source) as src,zipfile.ZipFile(target,'w') as out:
    for entry in src.infolist():
        name=entry.filename;leaf=name.removeprefix('META-INF/')
        if name.startswith('META-INF/') and '/' not in leaf and (leaf.upper()=='MANIFEST.MF' or leaf.upper().endswith(('.SF','.RSA','.DSA','.EC'))):continue
        out.writestr(entry,src.read(name))
print('UNSIGNED_ARTIFACT_CREATED')
