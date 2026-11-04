"""Подготовка закреплённых GeoIP/Geosite; проверка содержимого по SHA-256."""
import hashlib, lzma, shutil
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
        snapshot=root/'vendor_assets'/(name+'.db.xz')
        data=lzma.decompress(snapshot.read_bytes())
        if hashlib.sha256(data).hexdigest()!=expected:raise SystemExit('SHA-256 mismatch: '+name)
        shutil.copyfile(snapshot,output)
    (target/(name+'.version.txt')).write_text(version,encoding='ascii')
    print(name+': SHA-256 PASS')
