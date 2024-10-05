import json
from os import system

#system("curl https://www.oref.org.il/districts/cities_heb.json > districts.he.json-new")
#system("curl https://www.oref.org.il/districts/cities_eng.json > districts.en.json-new")
#system("curl https://www.oref.org.il/districts/cities_rus.json > districts.ru.json-new")
#system("curl https://www.oref.org.il/districts/cities_arb.json > districts.ar.json-new")

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

    for district in fix_data:
        did = district['id']
        if did in new_district_data[lang]:
            label = new_district_data[lang][did]['label'].split(' I ')[0]
            district['label'] = label
            district['label_he'] = new_district_data['he'][did]['label'].split(' I ')[0]

    for did, district in new_district_data[lang].items():
        if did not in missing_ids:
            continue

        fix_data.append({
            "label": district['label'].split(' I ')[0],
            "label_he": new_district_data['he'][did]['label'].split(' I ')[0],
            "value": district['cityAlId'],
            "id": did,
            "areaid": district['areaid'],
            "areaname": area_names.get(district['areaid'], ('', 0))[0],
            "migun_time": area_names.get(district['areaid'], ('', 0))[1],
        })

    with open(f"districts.{lang}.json", 'w', encoding='utf-8') as file:
        json.dump(fix_data, file, ensure_ascii=False, indent=2)
