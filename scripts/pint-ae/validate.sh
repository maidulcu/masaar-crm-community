#!/usr/bin/env bash
# Validates UBL invoices and credit notes against the Peppol PINT AE (UAE) rules:
# the UBL 2.1 XML schema, then the PINT core and UAE jurisdiction Schematron rules
# (published by OpenPeppol, packaged on Maven Central by the phive-rules project).
#
# Usage: scripts/pint-ae/validate.sh FILE.xml [FILE.xml ...]
# Needs: java, xmllint (libxml2-utils), python3, curl, unzip.
# Artifacts are downloaded once into $PINT_CACHE (default ~/.cache/masaar-pint-ae) and
# checked against pinned SHA-256 sums. Exits non-zero if any file has a fatal error.
set -euo pipefail

PINT_VERSION=2026.5
CACHE="${PINT_CACHE:-$HOME/.cache/masaar-pint-ae}"
MAVEN=https://repo1.maven.org/maven2
declare -A JARS=(
  [pint]="$MAVEN/com/helger/phive/rules/phive-rules-peppol-pint/4.6.3/phive-rules-peppol-pint-4.6.3.jar b9b7c6b03ff9c37e6846268e78675a26e477663c5f6e23f050c91a9bf18988a3"
  [ubl21]="$MAVEN/com/helger/ubl/ph-ubl21/10.2.1/ph-ubl21-10.2.1.jar 0bb13039b0f0983df9cf8c6f98b1f0f0c992bbd9ae2196766573e61607402405"
  [saxon]="$MAVEN/net/sf/saxon/Saxon-HE/10.9/Saxon-HE-10.9.jar 491d8edf4ec811d15c2b2417b007218b9b938f15e4dfbad004025beb4e70e960"
  [cct]="$MAVEN/com/helger/xsd/ph-xsds-ccts-cct-schemamodule/4.1.1/ph-xsds-ccts-cct-schemamodule-4.1.1.jar 85119e35f3239932aeef4f6a963cb35ba7ec03e486f5f33acb9203b39b0e8728"
  [xmldsig]="$MAVEN/com/helger/xsd/ph-xsds-xmldsig/4.1.1/ph-xsds-xmldsig-4.1.1.jar 1b5c1a2680cb254afc5032e3588604516e0a5a6856084d7349ef9c60fb073de5"
  [xades132]="$MAVEN/com/helger/xsd/ph-xsds-xades132/4.1.1/ph-xsds-xades132-4.1.1.jar cd39b78653ac86f5b37ecc6a94c64af0827725759794488cf8ba457bafc98d2f"
  [xades141]="$MAVEN/com/helger/xsd/ph-xsds-xades141/4.1.1/ph-xsds-xades141-4.1.1.jar 07980395ab4e5d86b03a952f05100073148d0e381bb0de05dabc5e6600106938"
)

mkdir -p "$CACHE"
for name in "${!JARS[@]}"; do
  read -r url sum <<<"${JARS[$name]}"
  jar="$CACHE/$name.jar"
  if [ ! -f "$jar" ] || ! echo "$sum  $jar" | sha256sum -c --quiet - 2>/dev/null; then
    curl -fsSL --retry 3 -o "$jar" "$url"
    echo "$sum  $jar" | sha256sum -c --quiet -
  fi
done
if [ ! -f "$CACHE/rules/.ready" ]; then
  rm -rf "$CACHE/rules"
  unzip -q -o "$CACHE/pint.jar" "external/schematron/pint-ae/$PINT_VERSION/billing/*" -d "$CACHE/rules"
  unzip -q -o "$CACHE/ubl21.jar" 'external/schemas/ubl21/*' -d "$CACHE/rules"
  common="$CACHE/rules/external/schemas/ubl21/common"
  for j in cct xmldsig xades132 xades141; do unzip -q -o -j "$CACHE/$j.jar" 'schemas/*.xsd' -d "$common"; done
  # The packaged UBL schemas import these namespaces without a location (the Java
  # validator resolves them); point the imports at the files for xmllint.
  python3 "$(dirname "$0")/link_imports.py" "$common"
  touch "$CACHE/rules/.ready"
fi
RULES="$CACHE/rules/external/schematron/pint-ae/$PINT_VERSION/billing"
XSD="$CACHE/rules/external/schemas/ubl21/maindoc"

status=0
for f in "$@"; do
  echo "== $f"
  root=$(python3 -c 'import sys,xml.etree.ElementTree as E; print(E.parse(sys.argv[1]).getroot().tag.split("}")[1])' "$f")
  case "$root" in
    Invoice) xsd="$XSD/UBL-Invoice-2.1.xsd" ;;
    CreditNote) xsd="$XSD/UBL-CreditNote-2.1.xsd" ;;
    *) echo "unknown document type $root"; status=1; continue ;;
  esac
  if ! xmllint --noout --schema "$xsd" "$f"; then status=1; continue; fi
  for xslt in PINT-UBL-validation-preprocessed.xslt PINT-jurisdiction-aligned-rules.xslt; do
    out=$(mktemp)
    if ! java -cp "$CACHE/saxon.jar" net.sf.saxon.Transform -s:"$f" -xsl:"$RULES/$xslt" -o:"$out" 2>"$out.log"; then
      cat "$out.log"; status=1; rm -f "$out" "$out.log"; continue
    fi
    rm -f "$out.log"
    if ! python3 "$(dirname "$0")/svrl.py" "$xslt" "$out"; then status=1; fi
    rm -f "$out"
  done
done
exit $status
