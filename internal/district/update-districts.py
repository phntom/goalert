import json
from os import system

system("curl https://www.oref.org.il/districts/cities_heb.json > districts.he.json-new")
system("curl https://www.oref.org.il/districts/cities_eng.json > districts.en.json-new")
system("curl https://www.oref.org.il/districts/cities_rus.json > districts.ru.json-new")
system("curl https://www.oref.org.il/districts/cities_arb.json > districts.ar.json-new")

# Load the JSON files
with open('districts.he.json', 'r', encoding='utf-8') as file:
    districts_data = json.load(file)

new_district_data = {}
for lang in ('he', 'en', 'ru', 'ar'):
    with open(f"districts.{lang}.json-new", 'r', encoding='utf-8') as file:
        new_district_data[lang] = {}
        for district in json.load(file):
            new_district_data[lang][district['id']] = district

scratch_data = new_district_data['he'].values()

district_ids_districts = {district['id'] for district in districts_data}
district_ids_scratch = {district['id'] for district in scratch_data}
missing_ids = district_ids_scratch - district_ids_districts

for lang in ('he', 'en', 'ru', 'ar'):
    with open(f"districts.{lang}.json", 'r', encoding='utf-8') as file:
        fix_data = json.load(file)

    area_names = {}
    with open(f'districts.{lang}.json', 'r', encoding='utf-8') as file:
        districts_data = json.load(file)
        for district in districts_data:
            area_names[district['areaid']] = district['areaname'], int(district['migun_time'])

    append_data = []
    for district in fix_data:
        did = district['id']
        if did in new_district_data[lang]:
            label = new_district_data[lang][did]['label'].split(' I ')[0]
            if ' | ' in label:
                label = label.split(' | ')[0]
            label = label.strip()

            if label != district['label']:
                print(f"Fixing {did} ({district['label']} -> {label}) from: {new_district_data[lang][did]['label']}")
                append_data.append({
                    "label": label,
                    "label_he": new_district_data['he'][did]['label'].split(' I ')[0],
                    "value": district['value'],
                    "id": did,
                    "areaid": district['areaid'],
                    "areaname": district['areaname'],
                    "migun_time": district['migun_time'],
                })

    fix_data = fix_data + append_data

    for did, district in new_district_data[lang].items():
        if did not in missing_ids:
            continue

        label = district['label'].split(' I ')[0]
        if ' | ' in label:
            label = label.split(' | ')[0]
        label = label.strip()

        fix_data.append({
            "label": label,
            "label_he": new_district_data['he'][did]['label'].split(' I ')[0].split(' | ')[0].strip(),
            "value": district['cityAlId'],
            "id": did,
            "areaid": district['areaid'],
            "areaname": area_names.get(district['areaid'], ('', 0))[0],
            "migun_time": area_names.get(district['areaid'], ('', 0))[1],
        })

    check_dup_ids = {}
    for district in fix_data:
        check_dup_ids[district['id']] = district['label']

    check_dups = {}
    for district in fix_data:
        check = district['label']
        if check in check_dups:
            version1 = check_dups[check]
            version2 = district
            id1 = version1['id']
            id2 = version2['id']
            if id1 == id2:
                print(f"Duplicate same ID: {check}")
                print(json.dumps(version1, ensure_ascii=False, indent=2))
                print(json.dumps(version2, ensure_ascii=False, indent=2))
                continue
            target1 = check_dup_ids.get(id1)
            target2 = check_dup_ids.get(id2)
            if target1 and not target2:
                print(f"Duplicate: {check} target2 does not exist in new values")
                district = version1
            elif target2 and not target1:
                print(f"Duplicate: {check} target1 does not exist in new values")
                district = version2
            elif version1['label'] == target1 and version2['label'] != target2:
                print(f"Duplicate: {check} version1 is more correct")
                district = version1
            elif version1['label'] != target1 and version2['label'] == target2:
                print(f"Duplicate: {check} version2 is more correct")
                district = version2
            elif version1['label'] == target1 and version2['label'] == target2:
                print(f"Duplicate: {check} both versions have same label, using lower label")
                district = version1 if int(version1['id']) < int(version2['id']) else version2
                print(json.dumps(version1, ensure_ascii=False, indent=2))
                print(json.dumps(target1, ensure_ascii=False, indent=2))
                print(json.dumps(version2, ensure_ascii=False, indent=2))
                print(json.dumps(target2, ensure_ascii=False, indent=2))
                continue
            else:
                print(f"Duplicate: {check}")
                print(json.dumps(version1, ensure_ascii=False, indent=2))
                print(json.dumps(version2, ensure_ascii=False, indent=2))
        check_dups[check] = district

    with open(f"districts.{lang}.json", 'w', encoding='utf-8') as file:
        json.dump(fix_data, file, ensure_ascii=False, indent=2)
