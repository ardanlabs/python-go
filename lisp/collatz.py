def collatz_step(n):
    if n % 2 == 0:
        return n // 2
    return n * 3 + 1


print(collatz_step(7))  # 22
