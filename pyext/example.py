from checksig import check_signature

root_dir = '/tmp/logs'

print(f'checking {root_dir!r}')
try:
    check_signature(root_dir)
    print('OK')
except ValueError as err:
    print(f'error: {err}')
