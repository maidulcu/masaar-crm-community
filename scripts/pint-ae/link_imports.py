"""Adds schemaLocation to the namespace-only <xsd:import>s of the packaged UBL schemas."""
import pathlib
import re
import sys

LOCATIONS = {
    "urn:un:unece:uncefact:data:specification:CoreComponentTypeSchemaModule:2": "CCTS_CCT_SchemaModule.xsd",
    "http://www.w3.org/2000/09/xmldsig#": "xmldsig-core-schema.xsd",
    "http://uri.etsi.org/01903/v1.3.2#": "XAdES01903v132-201601.xsd",
    "http://uri.etsi.org/01903/v1.4.1#": "XAdES01903v141-201601.xsd",
}
IMPORT = re.compile(r'<(\w+:)?import\b([^>]*?)/?>(</(\w+:)?import>)?', re.S)

for path in pathlib.Path(sys.argv[1]).glob("*.xsd"):
    src = path.read_text(encoding="utf-8")

    def fix(m):
        attrs = m.group(2)
        ns = re.search(r'namespace="([^"]+)"', attrs)
        if not ns or ns.group(1) not in LOCATIONS:
            return m.group(0)
        attrs = re.sub(r'\s*schemaLocation="[^"]*"', "", attrs)
        return f'<{m.group(1) or ""}import{attrs} schemaLocation="{LOCATIONS[ns.group(1)]}"/>'

    out = IMPORT.sub(fix, src)
    if out != src:
        path.write_text(out, encoding="utf-8")
