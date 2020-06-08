from checksig import check_signature

try:
    check_signature('/tmp/logs')
except ValueError as err:
    raise SystemExit(f'error: {err}')
