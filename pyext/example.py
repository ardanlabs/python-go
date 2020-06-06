from checksig import check_signature, SignatureError

try:
    check_signature('/tmp/logs')
except SignatureError as err:
    raise SystemExit(f'error: {err}')
