"""Prints the failed assertions of a Schematron (SVRL) report; exits 1 if any is fatal."""
import sys
import xml.etree.ElementTree as ET

NS = {"svrl": "http://purl.oclc.org/dsdl/svrl"}
name, path = sys.argv[1], sys.argv[2]
fatal = 0
for fa in ET.parse(path).getroot().iter("{http://purl.oclc.org/dsdl/svrl}failed-assert"):
    flag = fa.get("flag", "fatal")
    text = " ".join("".join(fa.find("svrl:text", NS).itertext()).split())
    print(f"  {flag.upper():7} {fa.get('id', '')}: {text}\n          at {fa.get('location', '')}")
    if flag == "fatal":
        fatal += 1
print(f"  {name}: {fatal} fatal")
sys.exit(1 if fatal else 0)
