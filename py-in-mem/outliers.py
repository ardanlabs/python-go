import numpy as np


def detect(data: np.ndarray):
    """Return indices where values more than 2 standard deviations from mean"""
    print(f'in : {data}')
    out = np.where(np.abs(data - data.mean()) > 2 * data.std())
    print(f'out:{out[0]}')
    # np.where returns a tuple for each dimension, we want the 1st element
    return out[0]
