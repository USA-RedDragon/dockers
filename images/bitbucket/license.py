import base64
import zlib
import struct
import sys
import time

from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import rsa, padding
from cryptography.hazmat.primitives.serialization import load_pem_private_key

import javaproperties

LICENSE_PREFIX = bytes([13, 14, 12, 10, 15])
SEPARATOR = 'X'
VERSION_SUFFIX = '02'

def base31_encode(num):
    if num == 0: return "0"
    alphabet = "0123456789abcdefghijklmnopqrstu"
    res = []
    while num > 0:
        res.append(alphabet[num % 31])
        num //= 31
    return "".join(reversed(res))

def pack_license(outer_private_key_path, hash_private_key_path, server_id, key_version):
    sen = str(int(time.time() * 1000) % 100000000).zfill(8)

    props = javaproperties.loads("")
    props.pop('licenseHash', None)
    props['keyVersion'] = key_version
    props['licenseVersion'] = "2"
    props['Subscription'] = "false"
    props['ServerID'] = server_id
    props['SEN'] = "SEN-L" + sen
    props['LicenseID'] = "LID" + props['SEN']
    props['Description'] = "Bitbucket (Data Center)"
    props['Evaluation'] = "false"
    props['stash.active'] = "true"
    props['stash.Starter'] = "false"
    props['stash.LicenseTypeName'] = "COMMERCIAL"
    props['stash.DataCenter'] = "true"
    props['Organisation'] = "Atlassian"
    props['ContactEMail'] = "bitbucket@atlassian.com"
    props['ContactName'] = "bitbucket@atlassian.com"
    props['stash.NumberOfUsers'] = "-1"
    props['CreationDate'] = time.strftime("%Y-%m-%d", time.gmtime())
    props['PurchaseDate'] = time.strftime("%Y-%m-%d", time.gmtime())
    props['MaintenanceExpiryDate'] = time.strftime("%Y-%m-%d", time.gmtime(time.time() + 100 * 365 * 24 * 3600))
    props['LicenseExpiryDate'] = time.strftime("%Y-%m-%d", time.gmtime(time.time() + 100 * 365 * 24 * 3600))
    serialized = _serialize_properties_sorted(props)

    with open(hash_private_key_path, 'rb') as key_file:
        hash_priv = load_pem_private_key(key_file.read(), password=None)
    if isinstance(hash_priv, rsa.RSAPrivateKey):
        signature_inner = hash_priv.sign(serialized.encode('utf-8'), padding.PKCS1v15(), hashes.SHA1())
    else:
        signature_inner = hash_priv.sign(serialized.encode('utf-8'), hashes.SHA1())
    props['licenseHash'] = base64.b64encode(signature_inner).decode('utf-8')

    properties_content_signed = _serialize_properties_sorted(props)

    payload_bytes = properties_content_signed.encode('utf-8')
    compressed_payload = zlib.compress(payload_bytes)
    license_text = LICENSE_PREFIX + compressed_payload

    with open(outer_private_key_path, 'rb') as key_file:
        private_key = load_pem_private_key(key_file.read(), password=None)

    if isinstance(private_key, rsa.RSAPrivateKey):
        signature = private_key.sign(license_text, padding.PKCS1v15(), hashes.SHA1())
    else:
        signature = private_key.sign(license_text, hashes.SHA1())

    packed_data = struct.pack('>I', len(license_text)) + license_text + signature

    raw_result = base64.b64encode(packed_data).decode('utf-8').strip()
    len_base31 = base31_encode(len(raw_result))
    final_string = f"{raw_result}{SEPARATOR}{VERSION_SUFFIX}{len_base31}"

    buf = []
    for i, char in enumerate(final_string):
        buf.append(char)
        if i > 0 and i % 76 == 0:
            buf.append('\n')

    return "".join(buf)

def _serialize_properties_sorted(props_dict):
    """Serialize properties in Java-sorted order matching SortedProperties without timestamp."""
    ordered = {k: props_dict[k] for k in sorted(props_dict.keys())}
    return javaproperties.dumps(ordered, comments=None, timestamp=False)

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage:", file=sys.stderr)
        print("  Generate: python license.py <outer_private.pem> <hash_private.pem> <server_id> [keyVersion]", file=sys.stderr)
        sys.exit(1)

    outer_key_path = sys.argv[1]
    hash_key_path = sys.argv[2]
    server_id = sys.argv[3]
    key_version = sys.argv[4] if len(sys.argv) > 4 else "1600708331"

    try:
        license_key = pack_license(outer_key_path, hash_key_path, server_id, key_version)
        print(license_key)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
