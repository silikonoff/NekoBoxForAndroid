"""Проверка независимой установки, русских материалов и запрета приватных файлов."""
from pathlib import Path
import re, subprocess, xml.etree.ElementTree as ET
root=Path(__file__).resolve().parents[1]
tracked=subprocess.check_output(['git','-C',str(root),'ls-files','-z']).decode().split('\0')
bad=[]
for name in filter(None,tracked):
    path=root/name
    if not path.exists():continue
    if re.search(r'\.(?:jks|keystore|pcap|pcapng|log)$',name,re.I) or name.startswith(('cache/','private/','artifacts/')):bad.append(name)
    if name in ('README.md','DOWNLOADS.md') or name.startswith('docs/'):
        if path.suffix=='.md' and re.search('[\u4e00-\u9fff]',path.read_text(encoding='utf-8')):bad.append(name+': CJK')
assert not bad,'Forbidden tracked files: '+str(bad)
assert 'PACKAGE_NAME=io.github.silikonoff.nekobox' in (root/'silikbox.properties').read_text()
manifest=(root/'app/src/main/AndroidManifest.xml').read_text()
assert '${applicationId}.SERVICE' in manifest
assert 'resourceConfigurations += listOf("ru", "en")' in (root/'buildSrc/src/main/kotlin/Helpers.kt').read_text()
for locale in ['values','values-ru']:
    values={e.get('name'):''.join(e.itertext()) for e in ET.parse(root/f'app/src/main/res/{locale}/strings.xml').getroot()}
    assert values['app_name']=='silikbox'
assert not list((root/'app/src/main/res').glob('values-zh*/strings.xml'))
for required in ['LICENSE','third_party/gomobile/LICENSE','third_party/libneko/go.mod','docs/AUTHORSHIP.md']:
    assert (root/required).is_file(),required
print('REPOSITORY_IDENTITY_RUSSIAN_AND_PRIVATE_PATHS=PASS')
