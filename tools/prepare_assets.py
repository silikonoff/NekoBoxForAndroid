"""Подготовка закреплённых GeoIP/Geosite; проверка содержимого по SHA-256."""
import argparse, hashlib, json, lzma, urllib.request
from pathlib import Path
root=Path(__file__).resolve().parents[1]
locks={
    'geoip':('SagerNet/sing-geoip','20260112','7b6348b4dad36a6b0c62f06f8f69cb00b2f438ee96dbae739f49d0ac387ad42a'),
    'geosite':('SagerNet/sing-geosite','20260208092911','33878593fe0af4a35093364909174520ae2351db6e4e8d317d4435ebf73d6cba'),
}
target=root/'app/src/main/assets/sing-box';target.mkdir(parents=True,exist_ok=True)
for name,(repo,version,expected) in locks.items():
    output=target/(name+'.db.xz')
    data=None
    if output.exists():
        try:data=lzma.decompress(output.read_bytes())
        except lzma.LZMAError:pass
    if data is None or hashlib.sha256(data).hexdigest()!=expected:
        url=f'https://github.com/{repo}/releases/download/{version}/{name}.db'
        with urllib.request.urlopen(url,timeout=120) as response:data=response.read()
        if hashlib.sha256(data).hexdigest()!=expected:raise SystemExit('SHA-256 mismatch: '+name)
        output.write_bytes(lzma.compress(data,preset=9))
    (target/(name+'.version.txt')).write_text(version,encoding='ascii')
    print(name+': SHA-256 PASS')
